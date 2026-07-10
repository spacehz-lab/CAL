from __future__ import annotations

import json
import re
import time
from pathlib import Path
from typing import Any

from constants import (
    STATUS_FAILED,
    STATUS_PASSED,
    STATUS_SKIPPED,
    STEP_DIRECT_REUSE_ORACLE,
    STEP_DIRECT_REUSE_RUN,
    STEP_INTENT_USE_ORACLE,
    STEP_INTENT_USE_RUN,
    STEP_INTENT_USE_SELECT,
)
from oracle import OracleRunner
from util import elapsed_ms, failure, flow_step, parse_json, template_inputs
from workspace import Workspace

USE_SELECTION_STRATEGY = "best"


class ReuseRunner:
    def __init__(self, bench: Path, home: Path, workspace: Workspace, oracle: OracleRunner) -> None:
        self.bench = bench
        self.home = home
        self.workspace = workspace
        self.oracle = oracle

    def run_direct_reuse(self, case: dict[str, Any], provider_result: dict[str, Any]) -> None:
        provider_id = provider_result.get("provider_id")
        if not provider_id:
            return
        for candidate in provider_result.get("candidates") or []:
            if not (candidate.get("promotion") or {}).get("binding_id"):
                continue
            for fixture in reuse_rounds(case):
                reuse = self.run_reuse_fixture(case, provider_id, candidate, fixture)
                candidate.setdefault("reuse", []).append(reuse)
                provider_result.setdefault("steps", []).extend(reuse.get("steps") or [])

    def run_intent_uses(self, case: dict[str, Any], case_result: dict[str, Any]) -> None:
        rounds = reuse_rounds(case)
        if not rounds:
            return
        for fixture in rounds:
            inputs = materialize_inputs(self.bench, self.home, case["id"], fixture)
            use = self.run_use_fixture(case, fixture, inputs)
            case_result.setdefault("use", []).append(use)

    def run_reuse_fixture(self, case: dict[str, Any], provider_id: str, candidate: dict[str, Any], fixture: dict[str, Any]) -> dict[str, Any]:
        inputs = materialize_inputs(self.bench, self.home, case["id"], fixture)
        run_inputs = runtime_inputs(case, inputs)
        missing_inputs = missing_runtime_inputs(candidate, run_inputs)
        if missing_inputs:
            skip = failure("reuse_skipped", "missing_runtime_inputs", ", ".join(missing_inputs))
            return {
                "fixture_id": fixture.get("id", ""),
                "status": STATUS_SKIPPED,
                "skip": skip,
                "inputs": summarize_inputs(self.home, run_inputs),
                "steps": [
                    flow_step(STEP_DIRECT_REUSE_RUN, STATUS_SKIPPED, fixture_id=fixture.get("id", ""), failure=skip),
                    flow_step(STEP_DIRECT_REUSE_ORACLE, STATUS_SKIPPED, fixture_id=fixture.get("id", ""), failure=skip),
                ],
            }
        started = time.monotonic()
        completed = self.workspace.run_calctl(
            [
                "runs",
                "create",
                "--capability-id",
                candidate.get("capability_id", ""),
                "--binding-id",
                (candidate.get("promotion") or {}).get("binding_id", ""),
                "--provider-id",
                provider_id,
                "--inputs-json",
                json.dumps(run_inputs, separators=(",", ":")),
                "--verify",
                "--json",
            ]
        )
        reuse: dict[str, Any] = {
            "fixture_id": fixture.get("id", ""),
            "duration_ms": elapsed_ms(started),
            "held_out": True,
            "inputs": summarize_inputs(self.home, run_inputs),
        }
        if completed.returncode != 0:
            reuse["status"] = STATUS_FAILED
            reuse["failure"] = command_failure("reuse_failed", completed)
            reuse["steps"] = [
                flow_step(STEP_DIRECT_REUSE_RUN, STATUS_FAILED, duration_ms=reuse["duration_ms"], fixture_id=fixture.get("id", ""), failure=reuse["failure"]),
                flow_step(STEP_DIRECT_REUSE_ORACLE, STATUS_SKIPPED, fixture_id=fixture.get("id", ""), failure=reuse["failure"]),
            ]
            return reuse
        reuse["run"] = parse_json(completed.stdout) or {}
        oracle = self.oracle.run(case, inputs)
        reuse["oracle"] = oracle
        reuse["status"] = STATUS_PASSED if oracle.get("passed") else STATUS_FAILED
        if not oracle.get("passed"):
            err = oracle.get("error") or {}
            reuse["failure"] = failure("oracle_failure", err.get("code", ""), err.get("message", ""))
        reuse["steps"] = [
            flow_step(STEP_DIRECT_REUSE_RUN, STATUS_PASSED, duration_ms=reuse["duration_ms"], fixture_id=fixture.get("id", ""), run_id=(reuse.get("run") or {}).get("id", "")),
            flow_step(STEP_DIRECT_REUSE_ORACLE, reuse["status"], fixture_id=fixture.get("id", ""), oracle_id=(oracle or {}).get("id", ""), failure=reuse.get("failure")),
        ]
        return reuse

    def run_use_fixture(self, case: dict[str, Any], fixture: dict[str, Any], inputs: dict[str, Any], provider_id: str = "") -> dict[str, Any]:
        intent = use_intent(case)
        call_inputs = use_inputs(case, inputs)
        args = [
            "use",
            intent,
            "--inputs-json",
            json.dumps(call_inputs, separators=(",", ":")),
            "--strategy",
            USE_SELECTION_STRATEGY,
            "--json",
        ]
        if provider_id:
            args.extend(["--provider-id", provider_id])
        started = time.monotonic()
        completed = self.workspace.run_calctl(args)
        use: dict[str, Any] = {
            "fixture_id": fixture.get("id", ""),
            "intent": intent,
            "provider_id": provider_id,
            "duration_ms": elapsed_ms(started),
            "held_out": True,
            "inputs": summarize_inputs(self.home, call_inputs),
        }
        if completed.returncode != 0:
            use["status"] = STATUS_FAILED
            use["failure"] = command_failure("use_failed", completed)
            use["steps"] = [
                flow_step(STEP_INTENT_USE_SELECT, STATUS_FAILED, fixture_id=fixture.get("id", ""), failure=use["failure"]),
                flow_step(STEP_INTENT_USE_RUN, STATUS_SKIPPED, fixture_id=fixture.get("id", ""), failure=use["failure"]),
                flow_step(STEP_INTENT_USE_ORACLE, STATUS_SKIPPED, fixture_id=fixture.get("id", ""), failure=use["failure"]),
            ]
            return use
        output = parse_json(completed.stdout)
        if not output:
            use["status"] = STATUS_FAILED
            use["failure"] = failure("use_failed", "invalid_use_json", "use did not return JSON")
            use["steps"] = [
                flow_step(STEP_INTENT_USE_SELECT, STATUS_FAILED, fixture_id=fixture.get("id", ""), failure=use["failure"]),
                flow_step(STEP_INTENT_USE_RUN, STATUS_SKIPPED, fixture_id=fixture.get("id", ""), failure=use["failure"]),
                flow_step(STEP_INTENT_USE_ORACLE, STATUS_SKIPPED, fixture_id=fixture.get("id", ""), failure=use["failure"]),
            ]
            return use
        if isinstance(output.get("error"), dict):
            err = output["error"]
            use["status"] = STATUS_FAILED
            use["failure"] = failure("use_failed", err.get("code", ""), err.get("message", ""))
            use["steps"] = [
                flow_step(STEP_INTENT_USE_SELECT, STATUS_FAILED, fixture_id=fixture.get("id", ""), failure=use["failure"]),
                flow_step(STEP_INTENT_USE_RUN, STATUS_SKIPPED, fixture_id=fixture.get("id", ""), failure=use["failure"]),
                flow_step(STEP_INTENT_USE_ORACLE, STATUS_SKIPPED, fixture_id=fixture.get("id", ""), failure=use["failure"]),
            ]
            return use
        if isinstance(output.get("duration_ms"), int):
            use["duration_ms"] = output["duration_ms"]
        use["selection"] = output.get("selection") or {}
        use["run"] = output.get("run") or {}
        if not use["run"]:
            use["status"] = STATUS_FAILED
            use["failure"] = failure("use_failed", "missing_run", "use did not return a run record")
            use["steps"] = [
                flow_step(STEP_INTENT_USE_SELECT, STATUS_FAILED, fixture_id=fixture.get("id", ""), failure=use["failure"]),
                flow_step(STEP_INTENT_USE_RUN, STATUS_SKIPPED, fixture_id=fixture.get("id", ""), failure=use["failure"]),
                flow_step(STEP_INTENT_USE_ORACLE, STATUS_SKIPPED, fixture_id=fixture.get("id", ""), failure=use["failure"]),
            ]
            return use
        oracle_inputs = use_oracle_inputs(inputs, output)
        oracle = self.oracle.run(case, oracle_inputs)
        use["oracle"] = oracle
        use["status"] = STATUS_PASSED if oracle.get("passed") else STATUS_FAILED
        if not oracle.get("passed"):
            err = oracle.get("error") or {}
            use["failure"] = failure("oracle_failure", err.get("code", ""), err.get("message", ""))
        use["steps"] = [
            flow_step(
                STEP_INTENT_USE_SELECT,
                STATUS_PASSED if use["selection"] else STATUS_FAILED,
                fixture_id=fixture.get("id", ""),
                capability_id=(use["selection"] or {}).get("capability_id", ""),
                binding_id=(use["selection"] or {}).get("binding_id", ""),
                provider_id=(use["selection"] or {}).get("provider_id", ""),
                source=(use["selection"] or {}).get("source", ""),
                shortlist_size=(use["selection"] or {}).get("shortlist_size", ""),
            ),
            flow_step(STEP_INTENT_USE_RUN, STATUS_PASSED, duration_ms=use["duration_ms"], fixture_id=fixture.get("id", ""), run_id=(use["run"] or {}).get("id", "")),
            flow_step(STEP_INTENT_USE_ORACLE, use["status"], fixture_id=fixture.get("id", ""), oracle_id=(oracle or {}).get("id", ""), failure=use.get("failure")),
        ]
        return use


def materialize_inputs(bench: Path, home: Path, case_id: str, fixture: dict[str, Any]) -> dict[str, Any]:
    work = home / "benchmark" / case_id / "reuse" / fixture.get("id", "fixture")
    inputs: dict[str, Any] = {}
    for key, value in (fixture.get("inputs") or {}).items():
        if not isinstance(value, str):
            inputs[key] = value
            continue
        if value.startswith("{work}/"):
            path = work / value[len("{work}/") :]
            path.parent.mkdir(parents=True, exist_ok=True)
            inputs[key] = str(path)
        elif value.startswith("fixtures/"):
            inputs[key] = str((bench / value).resolve())
        else:
            inputs[key] = value
    return inputs


def reuse_rounds(case: dict[str, Any]) -> list[dict[str, Any]]:
    return (case.get("reuse") or {}).get("rounds") or []


def runtime_inputs(case: dict[str, Any], inputs: dict[str, Any]) -> dict[str, Any]:
    cleaned = dict(inputs)
    expected_key = (case.get("oracle") or {}).get("expected_input") or "expected"
    cleaned.pop(expected_key, None)
    return cleaned


def use_inputs(case: dict[str, Any], inputs: dict[str, Any]) -> dict[str, Any]:
    cleaned = runtime_inputs(case, inputs)
    cleaned.pop("target", None)
    return cleaned


def use_oracle_inputs(inputs: dict[str, Any], output: dict[str, Any]) -> dict[str, Any]:
    oracle_inputs = dict(inputs)
    run_inputs = (output.get("run") or {}).get("inputs") or {}
    target = run_inputs.get("target")
    if target:
        oracle_inputs["target"] = target
    return oracle_inputs


def use_intent(case: dict[str, Any]) -> str:
    use = case.get("use") or {}
    return str(use.get("intent") or case.get("intent") or case.get("description") or case.get("id") or "")


def missing_runtime_inputs(candidate: dict[str, Any], inputs: dict[str, Any]) -> list[str]:
    return [name for name in required_runtime_inputs(candidate) if inputs.get(name) in ("", None)]


def required_runtime_inputs(candidate: dict[str, Any]) -> list[str]:
    execution = candidate.get("execution") or {}
    spec = execution.get("spec") or {}
    names: set[str] = set()
    for arg in spec.get("args") or []:
        if isinstance(arg, str):
            names.update(template_inputs(arg))
    stdout_path_input = spec.get("stdout_path_input")
    if isinstance(stdout_path_input, str) and stdout_path_input:
        names.add(stdout_path_input)
    return sorted(names)


def summarize_inputs(home: Path, inputs: dict[str, Any]) -> dict[str, str]:
    return {key: summarize_input_value(home, value) for key, value in sorted(inputs.items())}


def summarize_input_value(home: Path, value: Any) -> str:
    if not isinstance(value, str):
        return str(value)
    path = Path(value)
    if path.is_absolute():
        try:
            return str(path.relative_to(home))
        except ValueError:
            return path.name
    return value


def command_failure(stage: str, completed: Any) -> dict[str, str]:
    parsed = parse_json(completed.stdout)
    if parsed and isinstance(parsed.get("error"), dict):
        err = parsed["error"]
        return failure(stage, err.get("code", ""), err.get("message", ""))
    message = completed.stderr.strip() or completed.stdout.strip() or f"exit status {completed.returncode}"
    return failure(stage, "command_failed", message)


def provider_has_promoted_binding(provider: dict[str, Any]) -> bool:
    return any((candidate.get("promotion") or {}).get("binding_id") for candidate in provider.get("candidates") or [])


def provider_oracles_passed(provider: dict[str, Any]) -> bool:
    seen = False
    for candidate in provider.get("candidates") or []:
        for reuse in candidate.get("reuse") or []:
            if reuse.get("status") == STATUS_SKIPPED:
                continue
            seen = True
            if not (reuse.get("oracle") or {}).get("passed"):
                return False
    return seen
