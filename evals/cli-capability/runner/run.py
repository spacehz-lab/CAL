#!/usr/bin/env python3
from __future__ import annotations

import argparse
import hashlib
import html
import json
import os
import platform
import re
import shutil
import subprocess
import sys
import time
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


MODE_REPLAY = "replay"
MODE_LIVE_LLM = "live_llm"
STATUS_PASSED = "passed"
STATUS_FAILED = "failed"
STATUS_SKIPPED = "skipped"


def main() -> int:
    args = parse_args()
    repo = Path(__file__).resolve().parents[3]
    bench = Path(__file__).resolve().parents[1]
    task_catalog = TaskCatalog(bench)
    selected_tasks = task_catalog.select(args.tasks, args.level)
    model = os.environ.get("CAL_LLM_MODEL", "") if args.mode == MODE_LIVE_LLM else ""
    run_id = new_run_id(args.mode, model)
    out_base = Path(args.out).resolve() if args.out else repo / "evals" / "out" / "cli-capability"
    run_dir = out_base / run_id
    home = Path(args.home).resolve() if args.home else run_dir / "home"
    env = os.environ.copy()
    env["CAL_HOME"] = str(home)

    artifact = new_artifact(run_id, args, selected_tasks, model)
    writer = ArtifactWriter(run_dir, artifact)
    calctl = resolve_executable(args.calctl)
    cald = resolve_executable(args.cald)
    process = None
    try:
        if not args.no_start_cald:
            process = start_cald(cald, repo, env, run_dir)
        wait_for_cald(calctl, repo, env)
        runner = BenchmarkRunner(repo, bench, home, calctl, env, args.mode)
        for task in selected_tasks:
            print(f"cli-capability: starting {task['id']} mode={args.mode}", flush=True)
            result = runner.run_task(task)
            artifact["tasks"].append(result)
            update_metrics(artifact, args.mode)
            artifact["failure_taxonomy"] = failure_taxonomy(artifact["tasks"])
            writer.write()
            print(
                f"cli-capability: completed {task['id']} "
                f"providers={len(result.get('providers', []))} "
                f"promoted={count_promotions(result)} "
                f"direct_oracle_passed={count_oracle_passed(result)} "
                f"use_oracle_passed={count_use_oracle_passed(result)}",
                flush=True,
            )
        update_metrics(artifact, args.mode)
        artifact["capability_model"] = capability_model(artifact["tasks"])
        artifact["failure_taxonomy"] = failure_taxonomy(artifact["tasks"])
        artifact["status"] = "completed"
        writer.write()
        validate_result(args.mode, artifact)
        print(f"summary: {writer.summary_path}")
        print(f"html: {writer.html_path}")
        return 0
    finally:
        if process is not None:
            stop_process(process)


class TaskCatalog:
    def __init__(self, bench: Path) -> None:
        self.bench = bench
        self.tasks = read_jsonl(bench / "tasks.jsonl")
        self.providers = {provider["id"]: provider for provider in read_json(bench / "providers.json")["providers"]}

    def select(self, task_names: str, level: str = "") -> list[dict[str, Any]]:
        if task_names.strip():
            names = [name.strip() for name in task_names.split(",") if name.strip()]
        elif level:
            names = self.level_task_ids(level)
        else:
            names = ["file_hash_sha1"]
        by_id = {task["id"]: task for task in self.tasks}
        missing = [name for name in names if name not in by_id]
        if missing:
            raise SystemExit(f"unknown benchmark tasks: {', '.join(missing)}")
        selected = []
        for name in names:
            task = dict(by_id[name])
            missing = [provider_id for provider_id in task["provider_candidates"] if provider_id not in self.providers]
            if missing:
                raise SystemExit(f"{name}: unknown provider candidates: {', '.join(missing)}")
            task["providers"] = [self.providers[provider_id] for provider_id in task["provider_candidates"]]
            selected.append(task)
        return selected

    def level_task_ids(self, level: str) -> list[str]:
        if level == "full":
            return [task["id"] for task in self.tasks]
        names = [task["id"] for task in self.tasks if task.get("level") == level]
        if not names:
            raise SystemExit(f"no benchmark tasks for level {level}")
        return names


class BenchmarkRunner:
    def __init__(self, repo: Path, bench: Path, home: Path, calctl: str, env: dict[str, str], mode: str) -> None:
        self.repo = repo
        self.bench = bench
        self.home = home
        self.calctl = calctl
        self.env = env
        self.mode = mode

    def run_task(self, task: dict[str, Any]) -> dict[str, Any]:
        result: dict[str, Any] = {
            "id": task["id"],
            "intent": use_intent(task),
            "description": task.get("description", ""),
            "providers": [],
            "use": [],
        }
        for provider in task["providers"]:
            result["providers"].append(self.run_provider(task, provider))
        self.run_uses(task, result)
        return result

    def run_provider(self, task: dict[str, Any], provider: dict[str, Any]) -> dict[str, Any]:
        result: dict[str, Any] = {"id": provider["id"], "command": provider["command"]}
        provider_path = shutil.which(provider["command"])
        if not provider_path:
            result["status"] = STATUS_SKIPPED
            result["failure"] = failure("cli_unavailable", "cli_unavailable", f"{provider['command']} was not found on PATH")
            return result
        result["provider_path"] = provider_path

        proposal = self.replay_proposal_path(task["id"], provider["id"])
        if self.mode == MODE_REPLAY and not proposal.exists():
            result["status"] = STATUS_SKIPPED
            result["failure"] = failure("proposal_unavailable", "proposal_unavailable", f"missing replay proposal {proposal}")
            return result

        before = self.trace_ids()
        cmd = ["discovery", "run", "--provider-path", provider_path, "--json"]
        if self.mode == MODE_REPLAY:
            cmd.extend(["--proposal-path", str(proposal)])
        elif self.mode == MODE_LIVE_LLM:
            cmd.extend(["--mode", "llm"])
        started = time.monotonic()
        completed = run_command(self.calctl, cmd, self.repo, self.env)
        result["acquisition_duration_ms"] = elapsed_ms(started)
        if completed.stdout:
            result["discovery"] = parse_json(completed.stdout) or {}
            result["proposal_duration_ms"] = (result["discovery"] or {}).get("proposal_duration_ms", 0)
            if self.mode == MODE_LIVE_LLM:
                result["llm_duration_ms"] = result["proposal_duration_ms"]
        result["trace_id"] = (result.get("discovery") or {}).get("trace_id", "") or self.new_trace_id(before)
        self.add_trace_summary(result)
        if completed.returncode != 0:
            result["status"] = STATUS_FAILED
            result["failure"] = command_failure("acquisition_failed", completed)
            return result

        result["provider_id"] = first_provider_id(result.get("discovery") or {})
        if self.mode == MODE_REPLAY:
            self.run_reuse(task, result)
            result["status"] = STATUS_PASSED if provider_oracles_passed(result) else STATUS_FAILED
        else:
            result["status"] = STATUS_PASSED if provider_has_promoted_binding(result) else STATUS_FAILED
        if result["status"] == STATUS_FAILED and not result.get("failure"):
            result["failure"] = provider_failure(self.mode)
        return result

    def add_trace_summary(self, result: dict[str, Any]) -> None:
        trace_id = result.get("trace_id")
        if not trace_id:
            return
        path = self.home / "discovery" / trace_id / "trace.json"
        if not path.exists():
            return
        trace = read_json(path)
        result["observation_sources"] = [obs.get("source", "") for obs in trace.get("observations", [])]
        candidates = []
        for index, candidate in enumerate(trace.get("candidates", [])):
            candidates.append(
                {
                    "index": index,
                    "capability_id": candidate.get("capability_id", ""),
                    "description": candidate.get("description", ""),
                    "execution": candidate.get("execution", {}),
                    "probe": {"status": "not_run"},
                    "promotion": {},
                    "reuse": [],
                }
            )
        for probe in trace.get("probes", []):
            index = probe.get("candidate_index", -1)
            if 0 <= index < len(candidates):
                verifier = probe.get("verifier") or {}
                candidates[index]["verifier_id"] = verifier.get("id", "")
                candidates[index]["probe"] = {"status": "passed", "passed": True} if probe.get("passed") else {"status": "failed", "passed": False}
                if probe.get("error"):
                    err = probe["error"]
                    candidates[index]["probe"]["error"] = failure("probe_failed", err.get("code", ""), err.get("message", ""))
        for promotion in trace.get("promotions", []):
            index = promotion.get("candidate_index", -1)
            if 0 <= index < len(candidates):
                candidates[index]["promotion"] = {
                    "capability_action": promotion.get("capability_action", ""),
                    "binding_action": promotion.get("binding_action", ""),
                    "binding_id": promotion.get("binding_id", ""),
                }
        result["candidates"] = candidates

    def run_reuse(self, task: dict[str, Any], provider_result: dict[str, Any]) -> None:
        provider_id = provider_result.get("provider_id")
        if not provider_id:
            return
        for candidate in provider_result.get("candidates") or []:
            if not (candidate.get("promotion") or {}).get("binding_id"):
                continue
            for fixture in (task.get("reuse") or {}).get("fixtures") or []:
                reuse = self.run_reuse_fixture(task, provider_id, candidate, fixture)
                candidate.setdefault("reuse", []).append(reuse)

    def run_reuse_fixture(self, task: dict[str, Any], provider_id: str, candidate: dict[str, Any], fixture: dict[str, Any]) -> dict[str, Any]:
        inputs = materialize_inputs(self.bench, self.home, task["id"], fixture)
        run_inputs = runtime_inputs(task, inputs)
        missing_inputs = missing_runtime_inputs(candidate, run_inputs)
        if missing_inputs:
            return {
                "fixture_id": fixture.get("id", ""),
                "status": STATUS_SKIPPED,
                "skip": failure("reuse_skipped", "missing_runtime_inputs", ", ".join(missing_inputs)),
                "inputs": summarize_inputs(self.home, run_inputs),
            }
        started = time.monotonic()
        completed = run_command(
            self.calctl,
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
            ],
            self.repo,
            self.env,
        )
        reuse: dict[str, Any] = {
            "fixture_id": fixture.get("id", ""),
            "duration_ms": elapsed_ms(started),
            "inputs": summarize_inputs(self.home, run_inputs),
        }
        if completed.returncode != 0:
            reuse["status"] = STATUS_FAILED
            reuse["failure"] = command_failure("reuse_failed", completed)
            return reuse
        reuse["run"] = parse_json(completed.stdout) or {}
        oracle = self.run_oracle(task, inputs)
        reuse["oracle"] = oracle
        reuse["status"] = STATUS_PASSED if oracle.get("passed") else STATUS_FAILED
        if not oracle.get("passed"):
            err = oracle.get("error") or {}
            reuse["failure"] = failure("oracle_failure", err.get("code", ""), err.get("message", ""))
        return reuse

    def run_uses(self, task: dict[str, Any], task_result: dict[str, Any]) -> None:
        fixtures = (task.get("reuse") or {}).get("fixtures") or []
        if not fixtures:
            return
        for fixture in fixtures:
            inputs = materialize_inputs(self.bench, self.home, task["id"], fixture)
            use = self.run_use_fixture(task, fixture, inputs)
            task_result.setdefault("use", []).append(use)

    def run_use_fixture(self, task: dict[str, Any], fixture: dict[str, Any], inputs: dict[str, Any]) -> dict[str, Any]:
        intent = use_intent(task)
        call_inputs = use_inputs(task, inputs)
        started = time.monotonic()
        completed = run_command(
            self.calctl,
            [
                "use",
                intent,
                "--inputs-json",
                json.dumps(call_inputs, separators=(",", ":")),
                "--verify",
                "--json",
            ],
            self.repo,
            self.env,
        )
        use: dict[str, Any] = {
            "fixture_id": fixture.get("id", ""),
            "intent": intent,
            "duration_ms": elapsed_ms(started),
            "inputs": summarize_inputs(self.home, call_inputs),
        }
        if completed.returncode != 0:
            use["status"] = STATUS_FAILED
            use["failure"] = command_failure("use_failed", completed)
            return use
        output = parse_json(completed.stdout)
        if not output:
            use["status"] = STATUS_FAILED
            use["failure"] = failure("use_failed", "invalid_use_json", "use did not return JSON")
            return use
        use["selection"] = output.get("selection") or {}
        use["run"] = output.get("run") or {}
        oracle_inputs = use_oracle_inputs(inputs, output)
        oracle = self.run_oracle(task, oracle_inputs)
        use["oracle"] = oracle
        use["status"] = STATUS_PASSED if oracle.get("passed") else STATUS_FAILED
        if not oracle.get("passed"):
            err = oracle.get("error") or {}
            use["failure"] = failure("oracle_failure", err.get("code", ""), err.get("message", ""))
        return use

    def run_oracle(self, task: dict[str, Any], inputs: dict[str, Any]) -> dict[str, Any]:
        oracle = task["oracle"]
        oracle_path = self.bench / oracle["path"]
        request = {"inputs": inputs, "oracle": oracle}
        completed = subprocess.run(
            [sys.executable, str(oracle_path)],
            cwd=self.repo,
            input=json.dumps(request),
            text=True,
            capture_output=True,
        )
        parsed = parse_json(completed.stdout)
        if completed.returncode != 0 or not parsed:
            return {
                "passed": False,
                "error": {
                    "code": "oracle_failed",
                    "message": completed.stderr.strip() or completed.stdout.strip() or f"exit status {completed.returncode}",
                },
            }
        parsed["id"] = oracle.get("id", "")
        return parsed

    def replay_proposal_path(self, task_id: str, provider_id: str) -> Path:
        return self.bench / "proposals" / "replay" / task_id / f"{provider_id}.json"

    def trace_ids(self) -> set[str]:
        root = self.home / "discovery"
        if not root.exists():
            return set()
        return {path.name for path in root.iterdir() if path.is_dir()}

    def new_trace_id(self, before: set[str]) -> str:
        return next(iter(self.trace_ids() - before), "")


class ArtifactWriter:
    def __init__(self, run_dir: Path, artifact: dict[str, Any]) -> None:
        self.run_dir = run_dir
        self.artifact = artifact
        self.summary_path = run_dir / "summary.json"
        self.html_path = run_dir / "index.html"

    def write(self) -> None:
        clean = strip_private_fields(self.artifact)
        self.summary_path.parent.mkdir(parents=True, exist_ok=True)
        self.summary_path.write_text(json.dumps(clean, indent=2) + "\n", encoding="utf-8")
        self.html_path.write_text(render_html(clean), encoding="utf-8")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run the CLI Capability benchmark.")
    parser.add_argument("--mode", choices=[MODE_REPLAY, MODE_LIVE_LLM], default=MODE_REPLAY)
    parser.add_argument("--tasks", default="", help="comma-separated task ids; defaults to file_hash_sha1")
    parser.add_argument("--level", choices=["focus", "full"], default="", help="select benchmark tasks by level when --tasks is omitted")
    parser.add_argument("--calctl", default="calctl")
    parser.add_argument("--cald", default="cald")
    parser.add_argument("--home", default="")
    parser.add_argument("--out", default="")
    parser.add_argument("--no-start-cald", action="store_true")
    return parser.parse_args()


def new_artifact(run_id: str, args: argparse.Namespace, tasks: list[dict[str, Any]], model: str) -> dict[str, Any]:
    return {
        "run_id": run_id,
        "source": "cli-capability benchmark",
        "status": "running",
        "mode": args.mode,
        "level": args.level,
        "selected_tasks": [task["id"] for task in tasks],
        "goos": sys.platform,
        "goarch": platform.machine(),
        "llm": {
            "api": os.environ.get("CAL_LLM_API", "") if args.mode == MODE_LIVE_LLM else "",
            "model": model,
            "base_url_configured": bool(os.environ.get("CAL_LLM_BASE_URL")) if args.mode == MODE_LIVE_LLM else False,
        },
        "tasks": [],
        "summary": {},
        "scores": {},
        "capability_model": {},
        "failure_taxonomy": [],
    }


def update_metrics(artifact: dict[str, Any], mode: str) -> None:
    artifact["summary"] = summarize(artifact["tasks"], mode)
    artifact["scores"] = score(artifact["summary"], mode)


def materialize_inputs(bench: Path, home: Path, task_id: str, fixture: dict[str, Any]) -> dict[str, Any]:
    work = home / "benchmark" / task_id / "reuse" / fixture.get("id", "fixture")
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


def runtime_inputs(task: dict[str, Any], inputs: dict[str, Any]) -> dict[str, Any]:
    cleaned = dict(inputs)
    expected_key = (task.get("oracle") or {}).get("expected_input") or "expected"
    cleaned.pop(expected_key, None)
    return cleaned


def use_inputs(task: dict[str, Any], inputs: dict[str, Any]) -> dict[str, Any]:
    cleaned = runtime_inputs(task, inputs)
    cleaned.pop("target", None)
    return cleaned


def use_oracle_inputs(inputs: dict[str, Any], output: dict[str, Any]) -> dict[str, Any]:
    oracle_inputs = dict(inputs)
    run_inputs = (output.get("run") or {}).get("inputs") or {}
    target = run_inputs.get("target")
    if target:
        oracle_inputs["target"] = target
    return oracle_inputs


def use_intent(task: dict[str, Any]) -> str:
    use = task.get("use") or {}
    return str(use.get("intent") or task.get("intent") or task.get("description") or task.get("id") or "")


def missing_runtime_inputs(candidate: dict[str, Any], inputs: dict[str, Any]) -> list[str]:
    required = required_runtime_inputs(candidate)
    return [name for name in required if inputs.get(name) in ("", None)]


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


def template_inputs(value: str) -> list[str]:
    return re.findall(r"{{\s*([A-Za-z_][A-Za-z0-9_]*)\s*}}", value)


def summarize(tasks: list[dict[str, Any]], mode: str = MODE_REPLAY) -> dict[str, Any]:
    summary = {
        "task_attempted": len(tasks),
        "provider_attempted": 0,
        "provider_available": 0,
        "providers_with_promoted_bindings": 0,
        "candidate_count": 0,
        "probe_pass_count": 0,
        "probe_fail_count": 0,
        "promoted_bindings": 0,
        "held_out_reuses": 0,
        "skipped_reuses": 0,
        "held_out_uses": 0,
        "oracle_pass_count": 0,
        "oracle_fail_count": 0,
        "use_oracle_pass_count": 0,
        "use_oracle_fail_count": 0,
        "failed": 0,
        "provider_failures": 0,
        "candidate_negative_evidence": 0,
        "direct_reuse_failures": 0,
        "intent_use_failures": 0,
        "acquisition_duration_ms": 0,
        "reuse_duration_ms": 0,
        "use_duration_ms": 0,
        "llm_duration_ms": 0,
        "acquisition_llm_calls": 0,
        "run_stage_llm_calls": 0,
    }
    acquisition_count = reuse_count = use_count = llm_count = 0
    for task in tasks:
        for provider in task.get("providers") or []:
            summary["provider_attempted"] += 1
            if provider.get("provider_path"):
                summary["provider_available"] += 1
            if provider.get("failure"):
                summary["provider_failures"] += 1
                if mode == MODE_REPLAY:
                    summary["failed"] += 1
            if provider.get("acquisition_duration_ms") is not None:
                summary["acquisition_duration_ms"] += provider["acquisition_duration_ms"]
                acquisition_count += 1
            if provider.get("llm_duration_ms"):
                summary["llm_duration_ms"] += provider["llm_duration_ms"]
                summary["acquisition_llm_calls"] += 1
                llm_count += 1
            provider_promoted = False
            for candidate in provider.get("candidates") or []:
                summary["candidate_count"] += 1
                probe = candidate.get("probe") or {}
                if probe.get("passed"):
                    summary["probe_pass_count"] += 1
                elif probe.get("status") == "failed":
                    summary["probe_fail_count"] += 1
                    summary["candidate_negative_evidence"] += 1
                if (candidate.get("promotion") or {}).get("binding_id"):
                    summary["promoted_bindings"] += 1
                    provider_promoted = True
                for reuse in candidate.get("reuse") or []:
                    if reuse.get("status") == STATUS_SKIPPED:
                        summary["skipped_reuses"] += 1
                        continue
                    summary["held_out_reuses"] += 1
                    summary["reuse_duration_ms"] += reuse.get("duration_ms", 0)
                    reuse_count += 1
                    if (reuse.get("oracle") or {}).get("passed"):
                        summary["oracle_pass_count"] += 1
                    else:
                        summary["oracle_fail_count"] += 1
                        summary["direct_reuse_failures"] += 1
                        if mode == MODE_REPLAY:
                            summary["failed"] += 1
            if provider_promoted:
                summary["providers_with_promoted_bindings"] += 1
        for use in task.get("use") or []:
            summary["held_out_uses"] += 1
            summary["use_duration_ms"] += use.get("duration_ms", 0)
            use_count += 1
            if ((use.get("selection") or {}).get("source")) == "llm":
                summary["run_stage_llm_calls"] += 1
            if (use.get("oracle") or {}).get("passed"):
                summary["use_oracle_pass_count"] += 1
            else:
                summary["use_oracle_fail_count"] += 1
                summary["intent_use_failures"] += 1
                summary["failed"] += 1
    summary["avg_acquisition_ms"] = average(summary["acquisition_duration_ms"], acquisition_count)
    summary["avg_reuse_ms"] = average(summary["reuse_duration_ms"], reuse_count)
    summary["avg_use_ms"] = average(summary["use_duration_ms"], use_count)
    summary["avg_llm_ms"] = average(summary["llm_duration_ms"], llm_count)
    if summary["held_out_reuses"]:
        summary["oracle_reuse_success_rate"] = summary["oracle_pass_count"] / summary["held_out_reuses"] if summary["held_out_reuses"] else 0
    if summary["held_out_uses"]:
        summary["oracle_use_success_rate"] = summary["use_oracle_pass_count"] / summary["held_out_uses"]
    return summary


def score(summary: dict[str, Any], mode: str) -> dict[str, Any]:
    direct_reuse_rate = ratio(summary.get("oracle_pass_count", 0), summary.get("held_out_reuses", 0))
    use_rate = ratio(summary.get("use_oracle_pass_count", 0), summary.get("held_out_uses", 0))
    closed_loop_rate = min(direct_reuse_rate, use_rate) if mode == MODE_REPLAY else use_rate
    return {
        "profile": mode,
        "closed_loop_success_rate": closed_loop_rate,
        "intent_use_success_rate": use_rate,
        "direct_reuse_success_rate": direct_reuse_rate if mode == MODE_REPLAY else None,
        "provider_yield_rate": ratio(summary.get("providers_with_promoted_bindings", 0), summary.get("provider_available", 0)),
        "probe_pass_rate": ratio(summary.get("probe_pass_count", 0), summary.get("candidate_count", 0)),
        "promotion_yield": ratio(summary.get("promoted_bindings", 0), summary.get("probe_pass_count", 0)),
        "provider_failure_rate": ratio(summary.get("provider_failures", 0), summary.get("provider_available", 0)),
        "negative_evidence_count": summary.get("candidate_negative_evidence", 0),
        "negative_evidence_rate": ratio(summary.get("candidate_negative_evidence", 0), summary.get("candidate_count", 0)),
        "failed_count": summary.get("failed", 0),
        "provider_acquisition_total_ms": summary.get("acquisition_duration_ms", 0),
        "provider_acquisition_avg_ms": summary.get("avg_acquisition_ms", 0),
        "proposal_llm_total_ms": summary.get("llm_duration_ms", 0),
        "proposal_llm_avg_ms": summary.get("avg_llm_ms", 0),
        "acquisition_local_overhead_total_ms": max(
            summary.get("acquisition_duration_ms", 0) - summary.get("llm_duration_ms", 0),
            0,
        ),
        "intent_use_total_ms": summary.get("use_duration_ms", 0),
        "intent_use_avg_ms": summary.get("avg_use_ms", 0),
        "replay_direct_reuse_total_ms": summary.get("reuse_duration_ms", 0) if mode == MODE_REPLAY else None,
        "replay_direct_reuse_avg_ms": summary.get("avg_reuse_ms", 0) if mode == MODE_REPLAY else None,
    }


def capability_model(tasks: list[dict[str, Any]]) -> dict[str, Any]:
    providers: dict[str, set[str]] = {}
    capabilities: dict[str, set[str]] = {}
    for task in tasks:
        for provider in task.get("providers") or []:
            provider_key = provider.get("id", "")
            for candidate in provider.get("candidates") or []:
                if not (candidate.get("promotion") or {}).get("binding_id"):
                    continue
                cap = candidate.get("capability_id", "")
                if not cap:
                    continue
                providers.setdefault(provider_key, set()).add(cap)
                capabilities.setdefault(cap, set()).add(provider_key)
    return {
        "providers": {key: sorted(value) for key, value in sorted(providers.items())},
        "capabilities": {key: sorted(value) for key, value in sorted(capabilities.items())},
    }


def failure_taxonomy(tasks: list[dict[str, Any]]) -> list[dict[str, Any]]:
    counts: dict[tuple[str, str], int] = {}
    for task in tasks:
        for provider in task.get("providers") or []:
            record_failure(counts, provider.get("failure"))
            for candidate in provider.get("candidates") or []:
                record_failure(counts, (candidate.get("probe") or {}).get("error"))
                for reuse in candidate.get("reuse") or []:
                    record_failure(counts, reuse.get("failure"))
                    record_failure(counts, reuse.get("skip"))
        for use in task.get("use") or []:
            record_failure(counts, use.get("failure"))
    return [
        {"stage": stage, "code": code, "count": count}
        for (stage, code), count in sorted(counts.items())
    ]


def record_failure(counts: dict[tuple[str, str], int], value: Any) -> None:
    if not isinstance(value, dict):
        return
    stage = value.get("stage") or "unknown"
    code = value.get("code") or "unknown"
    counts[(stage, code)] = counts.get((stage, code), 0) + 1


def render_html(artifact: dict[str, Any]) -> str:
    return f"""<!doctype html>
<html>
<head>
<meta charset="utf-8">
<title>CLI Capability Benchmark</title>
<style>
body{{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;margin:24px;color:#222}}
.grid{{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:12px;margin:16px 0}}
.card{{border:1px solid #ddd;border-radius:8px;padding:12px;background:#fafafa}}
.primary .card{{background:#f6fbf7;border-color:#c9e4cf}}
.timing .card{{background:#f7f8fb}}
table{{border-collapse:collapse;width:100%;font-size:14px}}
td,th{{border:1px solid #ddd;padding:8px;vertical-align:top}}
th{{background:#f5f5f5;text-align:left}}
code{{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:12px}}
.section{{margin-top:28px}}
.pill{{display:inline-block;border-radius:999px;padding:2px 8px;background:#eee;font-size:12px}}
.ok{{color:#126b31;font-weight:600}}
.fail{{color:#9b1c1c;font-weight:600}}
.muted{{color:#666}}
.warn{{color:#8a5a00;font-weight:600}}
summary{{cursor:pointer;font-weight:600;margin:12px 0}}
</style>
</head>
<body>
<h1>CLI Capability Benchmark</h1>
{render_overview(artifact)}
{render_main_result(artifact)}
{render_timing(artifact)}
{render_task_results(artifact)}
{render_intent_use(artifact)}
{render_acquisition_evidence(artifact)}
{render_negative_evidence(artifact)}
{render_raw_details(artifact)}
</body>
</html>
"""


def render_metric_cards(metrics: dict[str, Any]) -> str:
    return "".join(
        f"<div class='card'>{escape(metric_label(key))}<br><strong>{escape(format_metric(key, value))}</strong></div>"
        for key, value in metrics.items()
    )


def render_overview(artifact: dict[str, Any]) -> str:
    llm = artifact.get("llm") or {}
    return f"""<section class="section">
<h2>Run Overview</h2>
<div class="grid">
{overview_card("Mode", artifact.get("mode", ""))}
{overview_card("Status", artifact.get("status", ""))}
{overview_card("Run ID", artifact.get("run_id", ""))}
{overview_card("Tasks", ", ".join(artifact.get("selected_tasks") or []))}
{overview_card("Model", llm.get("model") or "n/a")}
{overview_card("Platform", f"{artifact.get('goos', '')} / {artifact.get('goarch', '')}")}
</div>
</section>"""


def overview_card(label: str, value: Any) -> str:
    return f"<div class='card'>{escape(label)}<br><strong>{escape(value)}</strong></div>"


def summary_fraction(summary: dict[str, Any], numerator_key: str, denominator_key: str) -> str:
    if numerator_key not in summary or denominator_key not in summary:
        return "n/a"
    return fraction(summary.get(numerator_key), summary.get(denominator_key))


def render_main_result(artifact: dict[str, Any]) -> str:
    mode = artifact.get("mode", "")
    summary = artifact.get("summary") or {}
    scores = artifact.get("scores") or {}
    cards = [
        overview_card("Closed-loop success", format_metric("closed_loop_success_rate", scores.get("closed_loop_success_rate"))),
        overview_card("Intent use", summary_fraction(summary, "use_oracle_pass_count", "held_out_uses")),
        overview_card("Promoted bindings", summary.get("promoted_bindings", 0)),
        overview_card("Provider yield", format_metric("provider_yield_rate", scores.get("provider_yield_rate"))),
        overview_card("Negative evidence", summary.get("candidate_negative_evidence", 0)),
        overview_card("Failed closed loop", summary.get("failed", 0)),
    ]
    if mode == MODE_REPLAY:
        cards.insert(2, overview_card("Direct reuse", summary_fraction(summary, "oracle_pass_count", "held_out_reuses")))
    return f"""<section class="section primary">
<h2>Main Result</h2>
<div class="grid">{''.join(cards)}</div>
</section>"""


def render_timing(artifact: dict[str, Any]) -> str:
    scores = artifact.get("scores") or {}
    rows = [
        timing_row("Provider acquisition", scores.get("provider_acquisition_total_ms"), scores.get("provider_acquisition_avg_ms")),
        timing_row("Proposal LLM", scores.get("proposal_llm_total_ms"), scores.get("proposal_llm_avg_ms")),
        timing_row("Acquisition local overhead", scores.get("acquisition_local_overhead_total_ms"), None),
        timing_row("Intent use", scores.get("intent_use_total_ms"), scores.get("intent_use_avg_ms")),
    ]
    if artifact.get("mode") == MODE_REPLAY:
        rows.append(timing_row("Replay direct reuse", scores.get("replay_direct_reuse_total_ms"), scores.get("replay_direct_reuse_avg_ms")))
    return f"""<section class="section timing">
<h2>Workflow Timing</h2>
<table>
<tr><th>Flow</th><th>Total</th><th>Average</th></tr>
{''.join(rows)}
</table>
</section>"""


def timing_row(label: str, total: Any, average_value: Any) -> str:
    return f"<tr><td>{escape(label)}</td><td>{escape(format_duration_value(total))}</td><td>{escape(format_duration_value(average_value))}</td></tr>"


def render_task_results(artifact: dict[str, Any]) -> str:
    rows = []
    for task in artifact.get("tasks") or []:
        providers = task.get("providers") or []
        provider_count = len(providers)
        yielded = sum(1 for provider in providers if provider_has_promoted_binding(provider))
        promoted = sum(count_provider_promotions(provider) for provider in providers)
        negative = sum(count_provider_negative_evidence(provider) for provider in providers)
        rows.append(
            f"<tr><td><code>{escape(task.get('id', ''))}</code><br>{escape(task.get('intent', task.get('description', '')))}</td>"
            f"<td>{status(task_use_status(task))}<br>{escape(task_use_fraction(task))}</td>"
            f"<td>{yielded}/{provider_count}</td><td>{promoted}</td><td>{negative}</td></tr>"
        )
    return f"""<section class="section">
<h2>Task Results</h2>
<table>
<tr><th>Task</th><th>Intent use oracle</th><th>Provider yield</th><th>Promoted bindings</th><th>Negative evidence</th></tr>
{''.join(rows)}
</table>
</section>"""


def render_intent_use(artifact: dict[str, Any]) -> str:
    rows = []
    for task in artifact.get("tasks") or []:
        for use in task.get("use") or []:
            rows.append(use_row(task, use))
    return f"""<section class="section">
<h2>Intent Use</h2>
<table>
<tr><th>Task</th><th>Intent</th><th>Selection</th><th>Use</th><th>Oracle</th><th>Failure</th></tr>
{''.join(rows)}
</table>
</section>"""


def render_acquisition_evidence(artifact: dict[str, Any]) -> str:
    rows = []
    for task in artifact.get("tasks") or []:
        for provider in task.get("providers") or []:
            candidates = provider.get("candidates") or []
            if not candidates:
                rows.append(provider_row(task, provider, {}))
            for candidate in candidates:
                rows.append(provider_row(task, provider, candidate))
    return f"""<section class="section">
<h2>Acquisition Evidence</h2>
<table>
<tr><th>Task</th><th>Provider</th><th>Capability</th><th>Probe</th><th>Promotion</th><th>Held-out reuse</th><th>Oracle</th><th>Failure</th></tr>
{''.join(rows)}
</table>
</section>"""


def render_negative_evidence(artifact: dict[str, Any]) -> str:
    rows = []
    for task in artifact.get("tasks") or []:
        for provider in task.get("providers") or []:
            if provider.get("failure"):
                rows.append(negative_row(task, provider, {}, "provider_failed", provider.get("failure")))
            for candidate in provider.get("candidates") or []:
                probe_error = (candidate.get("probe") or {}).get("error")
                if probe_error:
                    rows.append(negative_row(task, provider, candidate, "probe_rejected", probe_error))
                for reuse in candidate.get("reuse") or []:
                    if reuse.get("failure"):
                        rows.append(negative_row(task, provider, candidate, "direct_reuse_rejected", reuse.get("failure")))
                    if reuse.get("skip"):
                        rows.append(negative_row(task, provider, candidate, "direct_reuse_skipped", reuse.get("skip")))
    if not rows:
        rows.append("<tr><td colspan='5'><span class='muted'>none</span></td></tr>")
    return f"""<section class="section">
<h2>Negative Evidence</h2>
<table>
<tr><th>Type</th><th>Task</th><th>Provider</th><th>Capability</th><th>Reason</th></tr>
{''.join(rows)}
</table>
</section>"""


def negative_row(task: dict[str, Any], provider: dict[str, Any], candidate: dict[str, Any], kind: str, failure_value: Any) -> str:
    return f"<tr><td><span class='warn'>{escape(kind)}</span></td><td><code>{escape(task.get('id', ''))}</code></td><td><code>{escape(provider.get('id', ''))}</code></td><td><code>{escape(candidate.get('capability_id', ''))}</code></td><td>{failure_text(failure_value if isinstance(failure_value, dict) else None)}</td></tr>"


def render_raw_details(artifact: dict[str, Any]) -> str:
    return f"""<section class="section">
<details>
<summary>Raw scores</summary>
<pre>{escape(json.dumps(artifact.get("scores", {}), indent=2))}</pre>
</details>
<details>
<summary>Raw summary</summary>
<pre>{escape(json.dumps(artifact.get("summary", {}), indent=2))}</pre>
</details>
<details>
<summary>Capability model</summary>
<pre>{escape(json.dumps(artifact.get("capability_model", {}), indent=2))}</pre>
</details>
<details>
<summary>Failure and negative evidence taxonomy</summary>
<pre>{escape(json.dumps(artifact.get("failure_taxonomy", []), indent=2))}</pre>
</details>
</section>"""


def metric_label(key: str) -> str:
    return key.replace("_", " ")


def format_metric(key: str, value: Any) -> str:
    if value is None:
        return "n/a"
    if isinstance(value, (int, float)) and (key.endswith("_rate") or key.endswith("_yield")):
        return f"{value * 100:.1f}%"
    if isinstance(value, int) and key.endswith("_ms"):
        return format_duration_ms(value)
    return str(value)


def format_duration_ms(value: int) -> str:
    if value >= 60_000:
        return f"{value / 60_000:.1f} min"
    if value >= 1_000:
        return f"{value / 1_000:.1f} s"
    return f"{value} ms"


def format_duration_value(value: Any) -> str:
    if value is None:
        return "n/a"
    if isinstance(value, int):
        return format_duration_ms(value)
    return str(value)


def fraction(numerator: Any, denominator: Any) -> str:
    numerator_value = int(numerator or 0)
    denominator_value = int(denominator or 0)
    if denominator_value == 0:
        return "0/0"
    return f"{numerator_value}/{denominator_value} ({ratio(numerator_value, denominator_value) * 100:.1f}%)"


def provider_row(task: dict[str, Any], provider: dict[str, Any], candidate: dict[str, Any]) -> str:
    reuse_items = candidate.get("reuse") or []
    reuse_text = "<br>".join(
        f"{escape(item.get('fixture_id'))}: {status(item.get('status'))} ({item.get('duration_ms', 0)} ms)"
        for item in reuse_items
    ) or "<span class='muted'>not run</span>"
    oracle_text = "<br>".join(
        f"{escape(item.get('fixture_id'))}: {status(oracle_status(item))}"
        for item in reuse_items
    ) or "<span class='muted'>not run</span>"
    promotion = candidate.get("promotion") or {}
    failure = provider.get("failure") or candidate_failure(candidate)
    return f"""<tr>
<td><code>{escape(task.get('id'))}</code><br>{escape(task.get('intent', task.get('description', '')))}</td>
<td><code>{escape(provider.get('id'))}</code><br>{escape(provider.get('provider_path', provider.get('command', '')))}</td>
<td><code>{escape(candidate.get('capability_id', ''))}</code><br>{escape(candidate.get('description', ''))}</td>
<td>{status((candidate.get('probe') or {}).get('status'))}</td>
<td>{escape(promotion.get('capability_action', ''))}/{escape(promotion.get('binding_action', ''))}<br><code>{escape(promotion.get('binding_id', ''))}</code></td>
<td>{reuse_text}</td>
<td>{oracle_text}</td>
<td>{failure_text(failure)}</td>
</tr>"""


def use_row(task: dict[str, Any], use: dict[str, Any]) -> str:
    selection = use.get("selection") or {}
    run = use.get("run") or {}
    oracle = use.get("oracle") or {}
    selected = "<br>".join(
        [
            f"capability: <code>{escape(selection.get('capability_id', ''))}</code>",
            f"binding: <code>{escape(selection.get('binding_id', ''))}</code>",
            f"provider: <code>{escape(selection.get('provider_id', ''))}</code>",
            f"source: {escape(selection.get('source', ''))}",
        ]
    )
    return f"""<tr>
<td><code>{escape(task.get('id'))}</code><br>{escape(task.get('intent', task.get('description', '')))}</td>
<td>{escape(use.get('intent', ''))}<br><code>{escape(use.get('fixture_id', ''))}</code></td>
<td>{selected}</td>
<td>{status(use.get('status'))} ({use.get('duration_ms', 0)} ms)<br><code>{escape(run.get('id', ''))}</code></td>
<td>{status('passed' if oracle.get('passed') else 'failed')}</td>
<td>{failure_text(use.get('failure'))}</td>
</tr>"""


def status(value: Any) -> str:
    value = str(value or "")
    if value == STATUS_PASSED or value == "passed":
        return "<span class='ok'>passed</span>"
    if value == STATUS_SKIPPED:
        return "<span class='muted'>skipped</span>"
    if value:
        return f"<span class='fail'>{escape(value)}</span>"
    return "<span class='muted'>none</span>"


def candidate_failure(candidate: dict[str, Any]) -> dict[str, str] | None:
    probe_error = (candidate.get("probe") or {}).get("error")
    if isinstance(probe_error, dict):
        return probe_error
    for reuse in candidate.get("reuse") or []:
        failed = reuse.get("failure")
        if isinstance(failed, dict):
            return failed
        skipped = reuse.get("skip")
        if isinstance(skipped, dict):
            return skipped
    return None


def oracle_status(item: dict[str, Any]) -> str:
    if item.get("status") == STATUS_SKIPPED:
        return STATUS_SKIPPED
    return STATUS_PASSED if (item.get("oracle") or {}).get("passed") else STATUS_FAILED


def failure_text(value: dict[str, str] | None) -> str:
    if not value:
        return "<span class='muted'>none</span>"
    parts = [value.get("stage", ""), value.get("code", ""), value.get("message", "")]
    return "<br>".join(f"<span class='fail'>{escape(part)}</span>" for part in parts if part)


def start_cald(cald: str, repo: Path, env: dict[str, str], run_dir: Path) -> subprocess.Popen[Any]:
    run_dir.mkdir(parents=True, exist_ok=True)
    log = open(run_dir / "cald.log", "ab")
    process = subprocess.Popen([cald, "serve"], cwd=repo, env=env, stdout=log, stderr=log)
    process._cal_log = log  # type: ignore[attr-defined]
    return process


def wait_for_cald(calctl: str, repo: Path, env: dict[str, str]) -> None:
    last = ""
    for _ in range(100):
        completed = run_command(calctl, ["daemon", "status", "--json"], repo, env)
        if completed.returncode == 0:
            status_doc = parse_json(completed.stdout) or {}
            if status_doc.get("running") is True:
                return
        last = completed.stderr.strip() or completed.stdout.strip()
        time.sleep(0.1)
    raise SystemExit(f"cald did not become ready: {last}")


def stop_process(process: subprocess.Popen[Any]) -> None:
    if process.poll() is None:
        process.terminate()
        try:
            process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait(timeout=5)
    log = getattr(process, "_cal_log", None)
    if log is not None:
        log.close()


def validate_result(mode: str, artifact: dict[str, Any]) -> None:
    summary = artifact.get("summary", {})
    if summary.get("use_oracle_pass_count", 0) < 1:
        raise SystemExit("benchmark produced no oracle-passing held-out use")
    if mode == MODE_REPLAY:
        if summary.get("oracle_pass_count", 0) < 1:
            raise SystemExit("benchmark produced no oracle-passing direct reuse")
        if summary.get("oracle_fail_count", 0) > 0:
            raise SystemExit("benchmark produced oracle failures")
        if summary.get("use_oracle_fail_count", 0) > 0:
            raise SystemExit("benchmark produced use oracle failures")
        if summary.get("failed", 0) > 0:
            raise SystemExit("replay benchmark produced failures")


def run_command(executable: str, args: list[str], repo: Path, env: dict[str, str]) -> subprocess.CompletedProcess[str]:
    return subprocess.run([executable, *args], cwd=repo, env=env, text=True, capture_output=True)


def resolve_executable(value: str) -> str:
    path = Path(value)
    if path.parent != Path(".") or path.is_absolute():
        return str(path.resolve())
    return shutil.which(value) or value


def first_provider_id(discovery: dict[str, Any]) -> str:
    providers = discovery.get("providers") or []
    return providers[0].get("id", "") if providers else ""


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


def command_failure(stage: str, completed: subprocess.CompletedProcess[str]) -> dict[str, str]:
    parsed = parse_json(completed.stdout)
    if parsed and isinstance(parsed.get("error"), dict):
        err = parsed["error"]
        return failure(stage, err.get("code", ""), err.get("message", ""))
    message = completed.stderr.strip() or completed.stdout.strip() or f"exit status {completed.returncode}"
    return failure(stage, "command_failed", message)


def failure(stage: str, code: str, message: str) -> dict[str, str]:
    return {"stage": stage, "code": code, "message": message}


def count_promotions(task: dict[str, Any]) -> int:
    count = 0
    for provider in task.get("providers") or []:
        count += count_provider_promotions(provider)
    return count


def count_provider_promotions(provider: dict[str, Any]) -> int:
    count = 0
    for candidate in provider.get("candidates") or []:
        if (candidate.get("promotion") or {}).get("binding_id"):
            count += 1
    return count


def count_provider_negative_evidence(provider: dict[str, Any]) -> int:
    count = 0
    if provider.get("failure"):
        count += 1
    for candidate in provider.get("candidates") or []:
        if (candidate.get("probe") or {}).get("error"):
            count += 1
        for reuse in candidate.get("reuse") or []:
            if reuse.get("failure") or reuse.get("skip"):
                count += 1
    return count


def task_use_status(task: dict[str, Any]) -> str:
    uses = task.get("use") or []
    if not uses:
        return ""
    return STATUS_PASSED if all((use.get("oracle") or {}).get("passed") for use in uses) else STATUS_FAILED


def task_use_fraction(task: dict[str, Any]) -> str:
    uses = task.get("use") or []
    passed = sum(1 for use in uses if (use.get("oracle") or {}).get("passed"))
    return fraction(passed, len(uses))


def count_oracle_passed(task: dict[str, Any]) -> int:
    count = 0
    for provider in task.get("providers") or []:
        count += count_oracle_passed_provider(provider)
    return count


def count_use_oracle_passed(task: dict[str, Any]) -> int:
    count = 0
    for use in task.get("use") or []:
        if (use.get("oracle") or {}).get("passed"):
            count += 1
    return count


def count_oracle_passed_provider(provider: dict[str, Any]) -> int:
    count = 0
    for candidate in provider.get("candidates") or []:
        for reuse in candidate.get("reuse") or []:
            if (reuse.get("oracle") or {}).get("passed"):
                count += 1
    return count


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


def provider_has_promoted_binding(provider: dict[str, Any]) -> bool:
    for candidate in provider.get("candidates") or []:
        if (candidate.get("promotion") or {}).get("binding_id"):
            return True
    return False


def provider_failure(mode: str) -> dict[str, str]:
    if mode == MODE_REPLAY:
        return failure("oracle_failure", "oracle_not_passed", "no held-out oracle passed")
    return failure("acquisition_failed", "no_promoted_binding", "no promoted binding")


def new_run_id(mode: str, model: str = "") -> str:
    now = datetime.now(timezone.utc)
    label = clean_run_part(model or mode)
    digest = hashlib.sha1(f"{now.isoformat()}|{mode}|{model}".encode("utf-8")).hexdigest()[:6]
    return f"{now.strftime('%Y%m%d-%H%M%S')}-{label}-{digest}"


def clean_run_part(value: str) -> str:
    cleaned = "".join(char.lower() if char.isalnum() else "-" for char in value.strip())
    cleaned = "-".join(part for part in cleaned.split("-") if part)
    return cleaned[:40] or "run"


def parse_json(text: str) -> dict[str, Any] | None:
    try:
        value = json.loads(text)
    except json.JSONDecodeError:
        return None
    return value if isinstance(value, dict) else None


def read_json(path: Path) -> dict[str, Any]:
    return json.loads(path.read_text(encoding="utf-8"))


def read_jsonl(path: Path) -> list[dict[str, Any]]:
    rows = []
    with path.open("r", encoding="utf-8") as handle:
        for line in handle:
            if line.strip():
                rows.append(json.loads(line))
    return rows


def strip_private_fields(value: Any) -> Any:
    if isinstance(value, dict):
        return {key: strip_private_fields(item) for key, item in value.items() if not key.startswith("_")}
    if isinstance(value, list):
        return [strip_private_fields(item) for item in value]
    return value


def elapsed_ms(started: float) -> int:
    return int((time.monotonic() - started) * 1000)


def average(total: int, count: int) -> int:
    return int(total / count) if count else 0


def ratio(numerator: int | float, denominator: int | float) -> float:
    return float(numerator) / float(denominator) if denominator else 0.0


def escape(value: Any) -> str:
    return html.escape(str(value if value is not None else ""))


if __name__ == "__main__":
    raise SystemExit(main())
