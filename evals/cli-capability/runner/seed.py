from __future__ import annotations

import hashlib
import json
import shutil
import time
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from constants import (
    STATUS_FAILED,
    STATUS_PASSED,
    STATUS_SKIPPED,
    STEP_PROVIDER_REGISTER,
    STEP_PROVIDER_RESOLVE,
)
from reuse import command_failure
from util import elapsed_ms, failure, flow_step, parse_json, read_json
from workspace import Workspace

REUSE_SEED_REPLAY = "replay"
REUSE_SEED_SELF = "self"
REUSE_SEEDS = [REUSE_SEED_REPLAY, REUSE_SEED_SELF]


class ReplaySeedRunner:
    def __init__(self, bench: Path, workspace: Workspace) -> None:
        self.bench = bench
        self.workspace = workspace

    def seed_case(self, case: dict[str, Any]) -> list[dict[str, Any]]:
        seeded: list[dict[str, Any]] = []
        for provider in case.get("providers") or []:
            seeded.append(self.seed_provider(case, provider))
        return seeded

    def seed_provider(self, case: dict[str, Any], provider: dict[str, Any]) -> dict[str, Any]:
        result: dict[str, Any] = {
            "id": provider["id"],
            "command": provider["command"],
            "provider_class": provider.get("provider_class", ""),
            "domains": provider.get("domains") or [],
            "optional": bool(provider.get("optional", False)),
            "seed": {"source": REUSE_SEED_REPLAY, "proposal_path": str(self.proposal_path(case["id"], provider["id"]))},
            "steps": [],
        }
        provider_path = shutil.which(provider["command"], path=self.workspace.env.get("PATH"))
        if not provider_path:
            result["status"] = STATUS_SKIPPED
            result["failure"] = failure("cli_unavailable", "cli_unavailable", f"{provider['command']} was not found on PATH")
            result["steps"].append(flow_step(STEP_PROVIDER_RESOLVE, STATUS_SKIPPED, failure=result["failure"]))
            return result
        result["provider_path"] = provider_path
        result["steps"].append(flow_step(STEP_PROVIDER_RESOLVE, STATUS_PASSED, provider_path=provider_path))

        proposal_path = self.proposal_path(case["id"], provider["id"])
        if not proposal_path.exists():
            result["status"] = STATUS_FAILED
            result["failure"] = failure("reuse_seed_failed", "proposal_unavailable", f"missing replay proposal {proposal_path}")
            return result

        registered = self.register_provider(provider_path)
        result["provider_register_duration_ms"] = registered.get("duration_ms", 0)
        if registered.get("provider"):
            result["provider_record"] = registered["provider"]
            result["provider_id"] = registered["provider"].get("id", "")
        if not result.get("provider_id"):
            provider_record = self.provider_by_path(provider_path)
            if provider_record:
                result["provider_record"] = provider_record
                result["provider_id"] = provider_record.get("id", "")
        result["steps"].append(registered["step"])
        if registered.get("failure"):
            result["status"] = STATUS_FAILED
            result["failure"] = registered["failure"]
            return result

        proposal = read_json(proposal_path)
        seeded = seed_capabilities(proposal, result["provider_id"])
        write_seeded_capabilities(self.workspace.home, seeded)
        result["candidates"] = seed_candidates(seeded)
        result["status"] = STATUS_PASSED if result["candidates"] else STATUS_FAILED
        if result["status"] == STATUS_FAILED:
            result["failure"] = failure("reuse_seed_failed", "no_seeded_candidates", "replay proposal did not produce seeded candidates")
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
        provider_id = provider.get("id", "")
        if not provider_id:
            return {"duration_ms": duration, "provider": provider, "step": flow_step(STEP_PROVIDER_REGISTER, STATUS_PASSED, duration_ms=duration)}
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

    def proposal_path(self, case_id: str, provider_id: str) -> Path:
        return self.bench / "proposals" / "replay" / case_id / f"{provider_id}.json"


def seed_capabilities(proposal: dict[str, Any], provider_id: str) -> list[dict[str, Any]]:
    capabilities: dict[str, dict[str, Any]] = {}
    probe_plans = {int(plan.get("candidate_index", -1)): plan for plan in proposal.get("probe_plans") or []}
    created_at = datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")
    for index, candidate in enumerate(proposal.get("candidates") or []):
        capability_id = candidate.get("capability_id", "")
        execution = candidate.get("execution") or {}
        verify = (probe_plans.get(index) or {}).get("verify") or {}
        if not capability_id or not execution or not verify:
            continue
        binding = {
            "id": binding_id(capability_id, provider_id, execution),
            "capability_id": capability_id,
            "provider_id": provider_id,
            "execution": execution,
            "verify": verify,
            "evidence": evidence_refs(verify),
            "state": "promoted",
            "created_at": created_at,
        }
        capability = capabilities.setdefault(
            capability_id,
            {"id": capability_id, "description": candidate.get("description", ""), "bindings": []},
        )
        capability["bindings"].append(binding)
    return list(capabilities.values())


def seed_candidates(capabilities: list[dict[str, Any]]) -> list[dict[str, Any]]:
    candidates: list[dict[str, Any]] = []
    for capability in capabilities:
        for binding in capability.get("bindings") or []:
            candidates.append(
                {
                    "capability_id": capability.get("id", ""),
                    "description": capability.get("description", ""),
                    "execution": binding.get("execution") or {},
                    "verification": binding.get("verify") or {},
                    "probe": {"status": STATUS_SKIPPED, "reason": "seeded_replay_verified"},
                    "promotion": {
                        "capability_action": "seeded",
                        "binding_action": "seeded",
                        "capability_id": capability.get("id", ""),
                        "binding_id": binding.get("id", ""),
                    },
                    "reuse": [],
                }
            )
    return candidates


def write_seeded_capabilities(home: Path, capabilities: list[dict[str, Any]]) -> None:
    root = home / "capabilities"
    root.mkdir(parents=True, exist_ok=True)
    for capability in capabilities:
        path = root / f"{capability['id']}.json"
        merged = merge_capability(read_json(path) if path.exists() else {}, capability)
        path.write_text(json.dumps(merged, indent=2) + "\n", encoding="utf-8")


def merge_capability(existing: dict[str, Any], incoming: dict[str, Any]) -> dict[str, Any]:
    if not existing:
        return incoming
    merged = dict(existing)
    if not merged.get("description"):
        merged["description"] = incoming.get("description", "")
    bindings = list(merged.get("bindings") or [])
    seen = {binding.get("id", "") for binding in bindings}
    for binding in incoming.get("bindings") or []:
        if binding.get("id", "") not in seen:
            bindings.append(binding)
            seen.add(binding.get("id", ""))
    merged["bindings"] = bindings
    return merged


def evidence_refs(verify: dict[str, Any]) -> list[dict[str, Any]]:
    refs: list[dict[str, Any]] = []
    for index, check in enumerate(verify.get("checks") or [], 1):
        predicate = check.get("predicate", "check")
        subject = check.get("subject") or {}
        label = subject.get("input") or subject.get("type") or "subject"
        refs.append(
            {
                "id": f"seed_check_{index}_{label}_{predicate}",
                "type": predicate,
                "content": {"predicate": predicate, "subject": subject},
            }
        )
    return refs or [{"id": "seed_replay_verified", "type": "replay_verified", "content": {"source": REUSE_SEED_REPLAY}}]


def binding_id(capability_id: str, provider_id: str, execution: dict[str, Any]) -> str:
    canonical = json.dumps(execution, separators=(",", ":"), sort_keys=True)
    digest = hashlib.sha256("|".join([capability_id, provider_id, canonical]).encode("utf-8")).hexdigest()[:12]
    return f"binding_{digest}"
