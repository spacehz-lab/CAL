from __future__ import annotations

import shutil
import time
from pathlib import Path
from typing import Any

from constants import (
    MODE_LIVE_LLM,
    MODE_REPLAY,
    STATUS_FAILED,
    STATUS_PASSED,
    STATUS_SKIPPED,
    STEP_ACQUISITION_RUN,
    STEP_PROVIDER_REGISTER,
    STEP_PROVIDER_RESOLVE,
    STEP_STAGE_BINDING,
    STEP_STAGE_CAPABILITY,
    STEP_STAGE_EVIDENCE,
    STEP_STAGE_OBSERVE,
    STEP_STAGE_PROBE,
    STEP_STAGE_PROMOTE,
    STEP_STAGE_SURFACE,
)
from reuse import command_failure, provider_oracles_passed, provider_has_promoted_binding
from util import elapsed_ms, failure, flow_step, parse_json, read_json
from workspace import Workspace


class AcquisitionRunner:
    def __init__(self, bench: Path, workspace: Workspace, mode: str) -> None:
        self.bench = bench
        self.workspace = workspace
        self.mode = mode

    def run_provider(self, case: dict[str, Any], provider: dict[str, Any]) -> dict[str, Any]:
        result: dict[str, Any] = {
            "id": provider["id"],
            "command": provider["command"],
            "provider_class": provider.get("provider_class", ""),
            "domains": provider.get("domains") or [],
            "optional": bool(provider.get("optional", False)),
            "steps": [],
        }
        provider_path = shutil.which(provider["command"])
        if not provider_path:
            result["status"] = STATUS_SKIPPED
            result["failure"] = failure("cli_unavailable", "cli_unavailable", f"{provider['command']} was not found on PATH")
            result["steps"].append(flow_step(STEP_PROVIDER_RESOLVE, STATUS_SKIPPED, failure=result["failure"]))
            return result
        result["provider_path"] = provider_path
        result["steps"].append(flow_step(STEP_PROVIDER_RESOLVE, STATUS_PASSED, provider_path=provider_path))

        proposal = self.replay_proposal_path(case["id"], provider["id"])
        if self.mode == MODE_REPLAY and not proposal.exists():
            result["status"] = STATUS_SKIPPED
            result["failure"] = failure("proposal_unavailable", "proposal_unavailable", f"missing replay proposal {proposal}")
            result["steps"].append(flow_step(STEP_ACQUISITION_RUN, STATUS_SKIPPED, failure=result["failure"]))
            return result

        registered = self.register_provider(provider_path)
        result["provider_register_duration_ms"] = registered.get("duration_ms", 0)
        if registered.get("provider"):
            result["provider_record"] = registered["provider"]
            result["provider_id"] = registered["provider"].get("id", "")
        result["steps"].append(registered["step"])
        if registered.get("failure"):
            result["status"] = STATUS_FAILED
            result["failure"] = registered["failure"]
            return result

        before = self.workspace.trace_ids()
        cmd = ["acquisition", "run", "--provider-id", result["provider_id"], "--hint", case["intent"], "--json"]
        if self.mode == MODE_REPLAY:
            cmd.extend(["--mode", MODE_REPLAY, "--proposal-path", str(proposal)])
        started = time.monotonic()
        completed = self.workspace.run_calctl(cmd)
        result["acquisition_duration_ms"] = elapsed_ms(started)
        if completed.stdout:
            result["acquisition"] = parse_json(completed.stdout) or {}
            result["proposal_duration_ms"] = (result["acquisition"] or {}).get("proposal_duration_ms", 0)
            if self.mode == MODE_LIVE_LLM:
                result["llm_duration_ms"] = result["proposal_duration_ms"]
        if completed.returncode != 0:
            result["status"] = STATUS_FAILED
            result["failure"] = command_failure("acquisition_failed", completed)
            result["steps"].append(flow_step(STEP_ACQUISITION_RUN, STATUS_FAILED, duration_ms=result["acquisition_duration_ms"], failure=result["failure"]))
            return result
        result["trace_id"] = (result.get("acquisition") or {}).get("trace_id", "") or self.workspace.new_trace_id(before)
        result["steps"].append(flow_step(STEP_ACQUISITION_RUN, STATUS_PASSED, duration_ms=result["acquisition_duration_ms"], trace_id=result.get("trace_id", "")))
        self.add_trace_summary(result)
        return result

    def register_provider(self, provider_path: str) -> dict[str, Any]:
        started = time.monotonic()
        completed = self.workspace.run_calctl(["providers", "add", "--provider-path", provider_path, "--json"])
        duration = elapsed_ms(started)
        if completed.returncode != 0:
            err = command_failure("provider_register_failed", completed)
            return {"duration_ms": duration, "failure": err, "step": flow_step(STEP_PROVIDER_REGISTER, STATUS_FAILED, duration_ms=duration, failure=err)}
        payload = parse_json(completed.stdout) or {}
        provider = payload.get("provider") if isinstance(payload.get("provider"), dict) else payload
        if not provider.get("id"):
            provider = self.provider_by_path(provider_path)
        provider_id = provider.get("id", "")
        if not provider_id:
            err = failure("provider_register_failed", "invalid_provider_json", "providers add did not return provider id")
            return {"duration_ms": duration, "failure": err, "step": flow_step(STEP_PROVIDER_REGISTER, STATUS_FAILED, duration_ms=duration, failure=err)}
        return {"duration_ms": duration, "provider": provider, "step": flow_step(STEP_PROVIDER_REGISTER, STATUS_PASSED, duration_ms=duration, provider_id=provider_id)}

    def provider_by_path(self, provider_path: str) -> dict[str, Any]:
        root = self.workspace.home / "providers"
        if not root.exists():
            return {}
        for path in root.glob("*.json"):
            provider = read_json(path)
            if provider.get("path") == provider_path:
                return provider
        return {}

    def add_trace_summary(self, result: dict[str, Any]) -> None:
        trace_id = result.get("trace_id")
        if not trace_id:
            return
        path = self.workspace.trace_path(trace_id)
        if not path.exists():
            return
        trace = read_json(path)
        result["observation_sources"] = [obs.get("source", "") for obs in trace.get("observations", [])]
        result["steps"].append(flow_step(STEP_STAGE_OBSERVE, STATUS_PASSED if result["observation_sources"] else STATUS_SKIPPED, sources=result["observation_sources"]))
        self.add_proposal_stage_steps(result, trace)
        candidates = self.trace_candidates(trace)
        self.add_probe_and_promotion(result, trace, candidates)
        result["candidates"] = candidates

    def add_proposal_stage_steps(self, result: dict[str, Any], trace: dict[str, Any]) -> None:
        proposal = trace.get("proposal") or {}
        stage_names = {
            "surface": STEP_STAGE_SURFACE,
            "capability": STEP_STAGE_CAPABILITY,
            "binding": STEP_STAGE_BINDING,
            "evidence": STEP_STAGE_EVIDENCE,
        }
        stages = {stage.get("name"): stage for stage in proposal.get("stages") or []}
        for name, step in stage_names.items():
            stage = stages.get(name) or {}
            summary = stage.get("summary") or {}
            selected = summary.get("selected", 0)
            raw = summary.get("raw", 0)
            result["steps"].append(flow_step(step, STATUS_PASSED if selected or raw else STATUS_SKIPPED, summary=summary, duration_ms=stage.get("duration_ms")))

    def trace_candidates(self, trace: dict[str, Any]) -> list[dict[str, Any]]:
        candidates = []
        for index, candidate in enumerate(trace.get("candidates", [])):
            candidates.append(
                {
                    "index": index,
                    "capability_id": candidate.get("capability_id", ""),
                    "description": candidate.get("description", ""),
                    "execution": candidate.get("execution", {}),
                    "probe": {"status": "not_run"},
                    "verification": {},
                    "promotion": {},
                    "reuse": [],
                }
            )
        return candidates

    def add_probe_and_promotion(self, result: dict[str, Any], trace: dict[str, Any], candidates: list[dict[str, Any]]) -> None:
        for probe in trace.get("probes", []):
            index = probe.get("candidate_index", -1)
            if 0 <= index < len(candidates):
                verify = probe.get("verify") or {}
                evidence = probe.get("evidence") or []
                candidates[index]["verification"] = {
                    "level": verify.get("level", ""),
                    "method": verify.get("method", ""),
                    "checks": verify.get("checks", []),
                    "evidence_count": len(evidence),
                }
                candidates[index]["probe"] = {"status": "passed", "passed": True} if probe.get("passed") else {"status": "failed", "passed": False}
                if probe.get("error"):
                    err = probe["error"]
                    candidates[index]["probe"]["error"] = failure("probe_failed", err.get("code", ""), err.get("message", ""))
        passed_probes = sum(1 for candidate in candidates if (candidate.get("probe") or {}).get("passed"))
        failed_probes = sum(1 for candidate in candidates if (candidate.get("probe") or {}).get("status") == STATUS_FAILED)
        probe_status = STATUS_PASSED if passed_probes else STATUS_FAILED if failed_probes else STATUS_SKIPPED
        result["steps"].append(flow_step(STEP_STAGE_PROBE, probe_status, passed=passed_probes, failed=failed_probes))
        for promotion in trace.get("promotions", []):
            index = promotion.get("candidate_index", -1)
            if 0 <= index < len(candidates):
                candidates[index]["promotion"] = {
                    "capability_action": promotion.get("capability_action", ""),
                    "binding_action": promotion.get("binding_action", ""),
                    "capability_id": promotion.get("capability_id", ""),
                    "binding_id": promotion.get("binding_id", ""),
                }
        promoted = sum(1 for candidate in candidates if (candidate.get("promotion") or {}).get("binding_id"))
        result["steps"].append(flow_step(STEP_STAGE_PROMOTE, STATUS_PASSED if promoted else STATUS_SKIPPED, promoted_bindings=promoted))

    def replay_proposal_path(self, case_id: str, provider_id: str) -> Path:
        return self.bench / "proposals" / "replay" / case_id / f"{provider_id}.json"


def finalize_provider_status(provider: dict[str, Any], mode: str, require_reuse_oracle: bool) -> None:
    if mode == MODE_REPLAY and require_reuse_oracle:
        provider["status"] = STATUS_PASSED if provider_oracles_passed(provider) else STATUS_FAILED
    else:
        provider["status"] = STATUS_PASSED if provider_has_promoted_binding(provider) else STATUS_FAILED
    if provider["status"] == STATUS_FAILED and not provider.get("failure"):
        provider["failure"] = provider_failure(mode, require_reuse_oracle)


def provider_failure(mode: str, require_reuse_oracle: bool) -> dict[str, str]:
    if mode == MODE_REPLAY and require_reuse_oracle:
        return failure("oracle_failure", "oracle_not_passed", "no held-out oracle passed")
    return failure("acquisition_failed", "no_promoted_binding", "no promoted binding")
