#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import os
import platform
import shutil
import subprocess
import sys
import time
from pathlib import Path
from typing import Any

import artifact as artifact_io
import report


MODE_REPLAY = "replay"
MODE_LIVE_LLM = "live_llm"


def main() -> int:
    args = parse_args()
    repo = Path(__file__).resolve().parents[3]
    exp_dir = Path(__file__).resolve().parent
    cases_doc = read_json(exp_dir / "cases.json")
    cases = select_cases(cases_doc, args.mode, args.level, args.cases)

    now = artifact_io.utc_now()
    model = os.environ.get("CAL_LLM_MODEL", "") if args.mode == MODE_LIVE_LLM else ""
    run_id = artifact_io.new_run_id(now, args.mode, model)
    out_base = Path(args.out).resolve() if args.out else artifact_io.default_out_dir(repo)
    run_dir = out_base / run_id
    home = Path(args.home).resolve() if args.home else run_dir / "home"
    env = os.environ.copy()
    env["CAL_HOME"] = str(home)
    artifact = new_artifact(run_id, args, cases, model)
    writer = ArtifactWriter(repo, exp_dir, run_dir, artifact)

    calctl = resolve_executable(args.calctl)
    cald = resolve_executable(args.cald)
    cald_process = None
    try:
        if not args.no_start_cald:
            cald_process = start_cald(cald, repo, env, run_dir)
        wait_for_cald(calctl, repo, env)
        runner = MatrixRunner(repo, exp_dir, home, calctl, env, args.mode)
        for index, case in enumerate(cases, 1):
            print(f"cli-matrix progress: starting {index}/{len(cases)} {case['name']} mode={args.mode}", flush=True)
            result = runner.run_case(case)
            artifact["cases"].append(result)
            artifact["summary"] = summarize(artifact["cases"])
            writer.write()
            print(
                "cli-matrix progress: completed "
                f"{index}/{len(cases)} {case['name']} candidates={len(result.get('candidates', []))} "
                f"promoted={promoted_count(result)} verified_reuses={verified_reuse_count(result)} "
                f"verified_uses={verified_use_count(result)} "
                f"failure={failure_summary(result.get('failure'))}",
                flush=True,
            )
        artifact["eval"] = run_json(calctl, ["eval", "--json"], repo, env)
        artifact["summary"] = summarize(artifact["cases"])
        artifact["status"] = "completed"
        writer.write()
        validate_result(args.mode, cases, artifact)
        print(f"summary: {writer.summary_path}")
        print(f"html: {writer.html_path}")
        return 0
    finally:
        if cald_process is not None:
            stop_process(cald_process)


class MatrixRunner:
    def __init__(self, repo: Path, exp_dir: Path, home: Path, calctl: str, env: dict[str, str], mode: str) -> None:
        self.repo = repo
        self.exp_dir = exp_dir
        self.home = home
        self.calctl = calctl
        self.env = env
        self.mode = mode

    def run_case(self, case: dict[str, Any]) -> dict[str, Any]:
        result: dict[str, Any] = {"name": case["name"], "cli": case["command"]}
        provider_path = shutil.which(case["command"])
        if not provider_path:
            result["failure"] = failure("cli_unavailable", "cli_unavailable", f"{case['command']} was not found on PATH")
            return result
        result["provider_path"] = provider_path
        before = self.trace_ids()
        cmd = ["discovery", "run", "--provider-path", provider_path, "--json"]
        if self.mode == MODE_REPLAY:
            proposal_path = self.exp_dir / case["replay_proposal"]
            cmd = ["discovery", "run", "--provider-path", provider_path, "--proposal-path", str(proposal_path), "--json"]
        started = time.monotonic()
        completed = run_command(self.calctl, cmd, self.repo, self.env)
        result["scan_duration_ms"] = elapsed_ms(started)
        stdout = completed.stdout
        if stdout:
            result["trace_id"] = (parse_json(stdout) or {}).get("trace_id", "")
        if not result.get("trace_id"):
            result["trace_id"] = self.new_trace_id(before)
        if completed.returncode != 0:
            result["failure"] = command_failure("proposal_failed", completed)
            self.add_trace_summary(result)
            better = failure_from_candidates(result)
            if better:
                result["failure"] = better
            return result

        acquisition = parse_json(stdout)
        if not acquisition:
            result["failure"] = failure("proposal_parse_failed", "invalid_discovery_json", "discovery did not return JSON")
            return result
        result["discovery_state"] = acquisition.get("state", "")
        result["trace_id"] = acquisition.get("trace_id", result.get("trace_id", ""))
        result["proposal_duration_ms"] = acquisition.get("proposal_duration_ms", 0)
        if self.mode == MODE_LIVE_LLM:
            result["llm_duration_ms"] = result["proposal_duration_ms"]
        providers = acquisition.get("providers") or []
        if providers:
            result["provider_id"] = providers[0].get("id", "")
        self.add_trace_summary(result)
        self.run_reuses(result)
        self.run_uses(result)
        return result

    def add_trace_summary(self, result: dict[str, Any]) -> None:
        trace_id = result.get("trace_id")
        if not trace_id:
            return
        trace_path = self.home / "discovery" / trace_id / "trace.json"
        if not trace_path.exists():
            return
        trace = read_json(trace_path)
        result["observation_sources"] = [obs.get("source", "") for obs in trace.get("observations", [])]
        candidates = []
        for index, candidate in enumerate(trace.get("candidates", [])):
            candidates.append(
                {
                    "index": index,
                    "capability_id": candidate.get("capability_id", ""),
                    "description": candidate.get("description", ""),
                    "execution": candidate.get("execution", {}),
                    "input_constraints": candidate.get("input_constraints", {}),
                    "probe": {"status": "not_run"},
                    "_probe_inputs_raw": {},
                }
            )
        for probe in trace.get("probes", []):
            index = probe.get("candidate_index", -1)
            if index < 0 or index >= len(candidates):
                continue
            candidate = candidates[index]
            verifier = probe.get("verifier") or {}
            candidate["verifier_id"] = verifier.get("id", "")
            candidate["verifier_source"] = self.verifier_source(candidate["verifier_id"])
            raw_inputs = probe.get("inputs") or {}
            candidate["_probe_inputs_raw"] = raw_inputs
            candidate["probe_inputs"] = summarize_inputs(self.home, raw_inputs)
            candidate["probe"] = {"status": "passed", "passed": True} if probe.get("passed") else {"status": "failed"}
            if probe.get("error"):
                err = probe["error"]
                failed = failure(probe_failure_stage(err.get("code", "")), err.get("code", ""), err.get("message", ""))
                candidate["probe"]["error"] = failed
                candidate["failure"] = failed
        for promotion in trace.get("promotions", []):
            index = promotion.get("candidate_index", -1)
            if index < 0 or index >= len(candidates):
                continue
            candidates[index]["promotion"] = {
                "capability_action": promotion.get("capability_action", ""),
                "binding_action": promotion.get("binding_action", ""),
                "binding_id": promotion.get("binding_id", ""),
            }
        result["candidates"] = candidates
        if trace.get("error") and not result.get("failure"):
            err = trace["error"]
            result["failure"] = failure(discovery_failure_stage(err.get("code", "")), err.get("code", ""), err.get("message", ""))

    def run_reuses(self, result: dict[str, Any]) -> None:
        provider_id = result.get("provider_id")
        if not provider_id:
            return
        for candidate in result.get("candidates", []):
            if not candidate.get("promotion") or not candidate.get("probe", {}).get("passed"):
                continue
            raw_inputs = candidate.get("_probe_inputs_raw") or {}
            if not raw_inputs:
                failed = failure("reuse_unavailable_input", "reuse_unavailable_input", "passed probe did not record reusable inputs")
                candidate["reuse"] = {"failure": failed}
                candidate["failure"] = failed
                continue
            started = time.monotonic()
            completed = run_command(
                self.calctl,
                [
                    "runs",
                    "create",
                    "--capability-id",
                    candidate["capability_id"],
                    "--provider-id",
                    provider_id,
                    "--inputs-json",
                    json.dumps(raw_inputs, separators=(",", ":")),
                    "--verify",
                    "--json",
                ],
                self.repo,
                self.env,
            )
            reuse: dict[str, Any] = {"duration_ms": elapsed_ms(started)}
            if completed.returncode != 0:
                failed = command_failure("reuse_execution_failed", completed)
                if failed["code"] == "verification_failed":
                    failed["stage"] = "reuse_verification_failed"
                reuse["failure"] = failed
                candidate["failure"] = failed
                candidate["reuse"] = reuse
                continue
            run_output = parse_json(completed.stdout)
            if not run_output:
                failed = failure("reuse_execution_failed", "invalid_run_json", "run did not return JSON")
                reuse["failure"] = failed
                candidate["failure"] = failed
                candidate["reuse"] = reuse
                continue
            reuse["status"] = run_output.get("status", "")
            reuse["verified"] = bool(run_output.get("verified"))
            reuse["evidence"] = run_output.get("evidence", [])
            candidate["reuse"] = reuse

    def run_uses(self, result: dict[str, Any]) -> None:
        for candidate in result.get("candidates", []):
            promotion = candidate.get("promotion") or {}
            if not promotion.get("binding_id") or not candidate.get("probe", {}).get("passed"):
                continue
            raw_inputs = candidate.get("_probe_inputs_raw") or {}
            intent = use_intent(result, candidate)
            use: dict[str, Any] = {"intent": intent, "inputs": summarize_inputs(self.home, use_inputs(raw_inputs))}
            if not raw_inputs:
                failed = failure("use_unavailable_input", "use_unavailable_input", "passed probe did not record reusable inputs")
                use["failure"] = failed
                candidate["use"] = use
                candidate["failure"] = failed
                continue
            started = time.monotonic()
            completed = run_command(
                self.calctl,
                [
                    "use",
                    intent,
                    "--inputs-json",
                    json.dumps(use_inputs(raw_inputs), separators=(",", ":")),
                    "--verify",
                    "--json",
                ],
                self.repo,
                self.env,
            )
            use["duration_ms"] = elapsed_ms(started)
            if completed.returncode != 0:
                failed = command_failure("use_execution_failed", completed)
                failed["stage"] = use_failure_stage(failed["code"])
                use["failure"] = failed
                candidate["failure"] = failed
                candidate["use"] = use
                continue
            use_output = parse_json(completed.stdout)
            if not use_output:
                failed = failure("use_execution_failed", "invalid_use_json", "use did not return JSON")
                use["failure"] = failed
                candidate["failure"] = failed
                candidate["use"] = use
                continue
            selection = use_output.get("selection") or {}
            run = use_output.get("run") or {}
            use["status"] = use_output.get("status", "")
            use["selection"] = {
                "source": selection.get("source", ""),
                "capability_id": selection.get("capability_id", ""),
                "binding_id": selection.get("binding_id", ""),
                "provider_id": selection.get("provider_id", ""),
                "candidates_considered": selection.get("candidates_considered", 0),
            }
            use["verified"] = bool(run.get("verified"))
            use["run_status"] = run.get("status", "")
            use["run_inputs"] = summarize_inputs(self.home, run.get("inputs") or {})
            use["evidence"] = run.get("evidence", [])
            if selection.get("binding_id") != promotion.get("binding_id"):
                failed = failure(
                    "use_selection_failed",
                    "wrong_binding",
                    f"use selected binding {selection.get('binding_id', '')}, want {promotion.get('binding_id', '')}",
                )
                use["failure"] = failed
                candidate["failure"] = failed
            elif not use["verified"]:
                failed = failure("use_verification_failed", "not_verified", "use run did not return verified evidence")
                use["failure"] = failed
                candidate["failure"] = failed
            candidate["use"] = use

    def trace_ids(self) -> set[str]:
        root = self.home / "discovery"
        if not root.exists():
            return set()
        return {path.name for path in root.iterdir() if path.is_dir()}

    def new_trace_id(self, before: set[str]) -> str:
        return next(iter(self.trace_ids() - before), "")

    def verifier_source(self, verifier_id: str) -> str:
        if verifier_id and (self.home / "verifiers" / verifier_id / "verify.py").exists():
            return "local"
        return "unknown"


class ArtifactWriter:
    def __init__(self, repo: Path, exp_dir: Path, run_dir: Path, artifact: dict[str, Any]) -> None:
        self.repo = repo
        self.exp_dir = exp_dir
        self.run_dir = run_dir
        self.artifact = artifact
        self.summary_path = run_dir / "summary.json"
        self.html_path = run_dir / "index.html"

    def write(self) -> None:
        clean = strip_private_fields(self.artifact)
        artifact_io.write_json(self.summary_path, clean)
        report.write_html(self.html_path, clean, self.exp_dir / "templates" / "index.html")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run the CAL CLI matrix experiment through calctl/cald.")
    parser.add_argument("--mode", choices=[MODE_REPLAY, MODE_LIVE_LLM], default=MODE_REPLAY)
    parser.add_argument("--level", choices=["smoke", "focus", "full"], default="")
    parser.add_argument("--cases", default="", help="comma-separated case names; overrides --level")
    parser.add_argument("--calctl", default="calctl")
    parser.add_argument("--cald", default="cald")
    parser.add_argument("--home", default="")
    parser.add_argument("--out", default="")
    parser.add_argument("--no-start-cald", action="store_true")
    return parser.parse_args()


def new_artifact(run_id: str, args: argparse.Namespace, cases: list[dict[str, Any]], model: str) -> dict[str, Any]:
    return {
        "run_id": run_id,
        "source": "cli matrix experiment",
        "status": "running",
        "mode": args.mode,
        "level": selected_level(args.mode, args.level, args.cases),
        "selected_cases": [case["name"] for case in cases],
        "goos": sys.platform,
        "goarch": platform.machine(),
        "llm": {
            "api": os.environ.get("CAL_LLM_API", ""),
            "model": model,
            "base_url_configured": bool(os.environ.get("CAL_LLM_BASE_URL")),
        },
        "cases": [],
        "summary": {},
    }


def select_cases(doc: dict[str, Any], mode: str, level: str, names: str) -> list[dict[str, Any]]:
    all_cases = {case["name"]: case for case in doc["cases"]}
    if names.strip():
        selected_names = unique_names(names.split(","))
    else:
        selected_names = doc["levels"][selected_level(mode, level, names)]
    missing = [name for name in selected_names if name not in all_cases]
    if missing:
        raise SystemExit(f"unknown cases: {', '.join(missing)}")
    selected = [all_cases[name] for name in selected_names]
    if not selected:
        raise SystemExit("no cases selected")
    return selected


def selected_level(mode: str, level: str, names: str) -> str:
    if names.strip():
        return "custom"
    if level:
        return level
    return "smoke" if mode == MODE_LIVE_LLM else "full"


def unique_names(names: list[str]) -> list[str]:
    result = []
    seen = set()
    for name in [part.strip() for part in names]:
        if name and name not in seen:
            result.append(name)
            seen.add(name)
    return result


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
            status = parse_json(completed.stdout) or {}
            if status.get("running") is True:
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


def run_json(executable: str, args: list[str], repo: Path, env: dict[str, str]) -> dict[str, Any]:
    completed = run_command(executable, args, repo, env)
    if completed.returncode != 0:
        raise SystemExit(command_failure("command_failed", completed)["message"])
    parsed = parse_json(completed.stdout)
    if parsed is None:
        raise SystemExit(f"{executable} {' '.join(args)} did not return JSON")
    return parsed


def run_command(executable: str, args: list[str], repo: Path, env: dict[str, str]) -> subprocess.CompletedProcess[str]:
    return subprocess.run([executable, *args], cwd=repo, env=env, text=True, capture_output=True)


def resolve_executable(value: str) -> str:
    path = Path(value)
    if path.parent != Path(".") or path.is_absolute():
        return str(path.resolve())
    resolved = shutil.which(value)
    return resolved or value


def summarize(cases: list[dict[str, Any]]) -> dict[str, Any]:
    summary = {
        "cli_attempted": len(cases),
        "cli_available": 0,
        "candidate_count": 0,
        "probe_pass_count": 0,
        "probe_fail_count": 0,
        "promoted_bindings": 0,
        "verified_reuses": 0,
        "intent_uses": 0,
        "verified_uses": 0,
        "use_fail_count": 0,
        "failed": 0,
        "llm_duration_ms": 0,
        "avg_scan_ms": 0,
        "avg_llm_ms": 0,
        "avg_run_ms": 0,
        "avg_use_ms": 0,
    }
    scan_total = scan_count = llm_count = run_total = run_count = use_total = use_count = 0
    for case in cases:
        if case.get("provider_path"):
            summary["cli_available"] += 1
        if case.get("failure"):
            summary["failed"] += 1
        if case.get("scan_duration_ms"):
            scan_total += case["scan_duration_ms"]
            scan_count += 1
        if case.get("llm_duration_ms"):
            summary["llm_duration_ms"] += case["llm_duration_ms"]
            llm_count += 1
        for candidate in case.get("candidates") or []:
            summary["candidate_count"] += 1
            probe = candidate.get("probe") or {}
            if probe.get("passed"):
                summary["probe_pass_count"] += 1
            elif probe.get("status") == "failed":
                summary["probe_fail_count"] += 1
            promotion = candidate.get("promotion") or {}
            if promotion.get("binding_id"):
                summary["promoted_bindings"] += 1
            reuse = candidate.get("reuse")
            if reuse:
                run_total += reuse.get("duration_ms", 0)
                run_count += 1
                if reuse.get("verified"):
                    summary["verified_reuses"] += 1
                if reuse.get("failure"):
                    summary["failed"] += 1
            use = candidate.get("use")
            if use:
                summary["intent_uses"] += 1
                use_total += use.get("duration_ms", 0)
                use_count += 1
                if use.get("verified"):
                    summary["verified_uses"] += 1
                if use.get("failure"):
                    summary["use_fail_count"] += 1
                    summary["failed"] += 1
            if candidate.get("failure") and not reuse:
                summary["failed"] += 1
    summary["avg_scan_ms"] = average(scan_total, scan_count)
    summary["avg_llm_ms"] = average(summary["llm_duration_ms"], llm_count)
    summary["avg_run_ms"] = average(run_total, run_count)
    summary["avg_use_ms"] = average(use_total, use_count)
    if summary["promoted_bindings"]:
        summary["reuse_success_rate"] = summary["verified_reuses"] / summary["promoted_bindings"]
        summary["use_success_rate"] = summary["verified_uses"] / summary["promoted_bindings"]
    return summary


def validate_result(mode: str, cases: list[dict[str, Any]], artifact: dict[str, Any]) -> None:
    verified = artifact.get("summary", {}).get("verified_reuses", 0)
    verified_uses = artifact.get("summary", {}).get("verified_uses", 0)
    if mode == MODE_REPLAY and verified < len(cases):
        raise SystemExit(f"verified reuses = {verified}, want {len(cases)} replay reuses")
    if mode == MODE_REPLAY and verified_uses < len(cases):
        raise SystemExit(f"verified uses = {verified_uses}, want {len(cases)} replay uses")
    if mode == MODE_LIVE_LLM and verified < 1:
        raise SystemExit("live_llm produced no verified reuse")
    if mode == MODE_LIVE_LLM and verified_uses < 1:
        raise SystemExit("live_llm produced no verified intent use")


def use_intent(case: dict[str, Any], candidate: dict[str, Any]) -> str:
    description = str(candidate.get("description") or "").strip()
    if description:
        return description
    configured = str(case.get("use_intent") or "").strip()
    if configured:
        return configured
    capability_id = str(candidate.get("capability_id") or "").replace(".", " ").replace("_", " ").strip()
    return capability_id or f"use {case.get('cli', 'provider')}"


def use_inputs(inputs: dict[str, Any]) -> dict[str, Any]:
    return {key: value for key, value in inputs.items() if key != "target"}


def summarize_inputs(home: Path, inputs: dict[str, Any]) -> list[dict[str, str]]:
    return [{"key": key, "value": summarize_input_value(home, inputs[key])} for key in sorted(inputs)]


def summarize_input_value(home: Path, value: Any) -> str:
    if not isinstance(value, str):
        return str(value)
    path = Path(value)
    if path.is_absolute():
        try:
            return str(path.relative_to(home))
        except ValueError:
            return path.name
    return value[:77] + "..." if len(value) > 80 else value


def command_failure(stage: str, completed: subprocess.CompletedProcess[str]) -> dict[str, str]:
    parsed = parse_json(completed.stdout)
    if parsed and isinstance(parsed.get("error"), dict):
        err = parsed["error"]
        return failure(stage, err.get("code", ""), err.get("message", ""))
    message = completed.stderr.strip() or completed.stdout.strip() or f"exit status {completed.returncode}"
    return failure(stage, "command_failed", message)


def failure(stage: str, code: str, message: str) -> dict[str, str]:
    return {"stage": stage, "code": code, "message": message}


def failure_from_candidates(result: dict[str, Any]) -> dict[str, str] | None:
    for candidate in result.get("candidates") or []:
        if candidate.get("failure"):
            return candidate["failure"]
    return result.get("failure")


def discovery_failure_stage(code: str) -> str:
    if code == "candidate_proposal_failed":
        return "proposal_failed"
    if code == "verification_failed":
        return "probe_verification_failed"
    if code == "promotion_failed":
        return "promotion_failed"
    if code == "observation_failed":
        return "observation_failed"
    return "proposal_failed"


def probe_failure_stage(code: str) -> str:
    if code == "execution_failed":
        return "probe_execution_failed"
    if code == "verification_failed":
        return "probe_verification_failed"
    return "probe_plan_failed"


def use_failure_stage(code: str) -> str:
    if code in {"no_match", "ambiguous", "llm_selection_failed", "invalid_llm_selection"}:
        return "use_selection_failed"
    if code in {"missing_inputs", "artifact_path_failed", "invalid_use_input"}:
        return "use_input_failed"
    if code == "verification_failed":
        return "use_verification_failed"
    return "use_execution_failed"


def promoted_count(result: dict[str, Any]) -> int:
    return sum(1 for candidate in result.get("candidates") or [] if (candidate.get("promotion") or {}).get("binding_id"))


def verified_reuse_count(result: dict[str, Any]) -> int:
    return sum(1 for candidate in result.get("candidates") or [] if (candidate.get("reuse") or {}).get("verified"))


def verified_use_count(result: dict[str, Any]) -> int:
    return sum(1 for candidate in result.get("candidates") or [] if (candidate.get("use") or {}).get("verified"))


def failure_summary(value: dict[str, str] | None) -> str:
    if not value:
        return ""
    return ": ".join(part for part in [value.get("stage"), value.get("code"), value.get("message")] if part)


def parse_json(text: str) -> dict[str, Any] | None:
    try:
        value = json.loads(text)
    except json.JSONDecodeError:
        return None
    return value if isinstance(value, dict) else None


def read_json(path: Path) -> dict[str, Any]:
    return json.loads(path.read_text(encoding="utf-8"))


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


if __name__ == "__main__":
    raise SystemExit(main())
