#!/usr/bin/env python3
from __future__ import annotations

import argparse
import hashlib
import html
import json
import os
import platform
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
    repo = Path(__file__).resolve().parents[4]
    bench = Path(__file__).resolve().parents[1]
    task_catalog = TaskCatalog(bench)
    selected_tasks = task_catalog.select(args.tasks, args.level)
    model = os.environ.get("CAL_LLM_MODEL", "") if args.mode == MODE_LIVE_LLM else ""
    run_id = new_run_id(args.mode, model)
    out_base = Path(args.out).resolve() if args.out else repo / "evals" / "out" / "benchmarks" / "cal-cli-v0"
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
            print(f"cal-cli-v0: starting {task['id']} mode={args.mode}", flush=True)
            result = runner.run_task(task)
            artifact["tasks"].append(result)
            artifact["summary"] = summarize(artifact["tasks"])
            artifact["failure_taxonomy"] = failure_taxonomy(artifact["tasks"])
            writer.write()
            print(
                f"cal-cli-v0: completed {task['id']} "
                f"providers={len(result.get('providers', []))} "
                f"promoted={count_promotions(result)} "
                f"oracle_passed={count_oracle_passed(result)}",
                flush=True,
            )
        artifact["summary"] = summarize(artifact["tasks"])
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
            "capability_goal": task["capability_goal"],
            "description": task.get("description", ""),
            "providers": [],
        }
        for provider in task["providers"]:
            result["providers"].append(self.run_provider(task, provider))
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
        self.run_reuse(task, result)
        result["status"] = STATUS_PASSED if provider_oracles_passed(result) else STATUS_FAILED
        if result["status"] == STATUS_FAILED and not result.get("failure"):
            result["failure"] = failure("oracle_failure", "oracle_not_passed", "no held-out oracle passed")
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
                    "input_constraints": candidate.get("input_constraints", {}),
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
            if not candidate_matches_task(task, candidate):
                continue
            for fixture in (task.get("reuse") or {}).get("fixtures") or []:
                reuse = self.run_reuse_fixture(task, provider_result["id"], provider_id, candidate["capability_id"], fixture)
                candidate.setdefault("reuse", []).append(reuse)

    def run_reuse_fixture(self, task: dict[str, Any], provider_name: str, provider_id: str, capability_id: str, fixture: dict[str, Any]) -> dict[str, Any]:
        inputs = materialize_inputs(self.bench, self.home, task["id"], fixture, input_overrides(task, provider_name, capability_id))
        started = time.monotonic()
        completed = run_command(
            self.calctl,
            [
                "runs",
                "create",
                "--capability-id",
                capability_id,
                "--provider-id",
                provider_id,
                "--inputs-json",
                json.dumps(inputs, separators=(",", ":")),
                "--verify",
                "--json",
            ],
            self.repo,
            self.env,
        )
        reuse: dict[str, Any] = {
            "fixture_id": fixture.get("id", ""),
            "duration_ms": elapsed_ms(started),
            "inputs": summarize_inputs(self.home, inputs),
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
    parser = argparse.ArgumentParser(description="Run the CAL CLI v0 benchmark.")
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
        "source": "cal-cli-v0 benchmark",
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
        "capability_model": {},
        "failure_taxonomy": [],
    }


def materialize_inputs(bench: Path, home: Path, task_id: str, fixture: dict[str, Any], overrides: dict[str, Any] | None = None) -> dict[str, Any]:
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
    for key, value in (overrides or {}).items():
        inputs[key] = value
    return inputs


def candidate_matches_task(task: dict[str, Any], candidate: dict[str, Any]) -> bool:
    capability_id = candidate.get("capability_id")
    accepted = set(task.get("accepted_capability_ids") or [])
    accepted.add(task["capability_goal"])
    return capability_id in accepted


def input_overrides(task: dict[str, Any], provider_name: str, capability_id: str) -> dict[str, Any]:
    overrides = ((task.get("reuse") or {}).get("input_overrides") or {}).get(capability_id) or {}
    provider_overrides = overrides.get(provider_name) or {}
    return dict(provider_overrides)


def summarize(tasks: list[dict[str, Any]]) -> dict[str, Any]:
    summary = {
        "task_attempted": len(tasks),
        "provider_attempted": 0,
        "provider_available": 0,
        "candidate_count": 0,
        "probe_pass_count": 0,
        "probe_fail_count": 0,
        "promoted_bindings": 0,
        "held_out_reuses": 0,
        "oracle_pass_count": 0,
        "oracle_fail_count": 0,
        "failed": 0,
        "acquisition_duration_ms": 0,
        "reuse_duration_ms": 0,
        "llm_duration_ms": 0,
        "acquisition_llm_calls": 0,
        "run_stage_llm_calls": 0,
    }
    acquisition_count = reuse_count = llm_count = 0
    for task in tasks:
        for provider in task.get("providers") or []:
            summary["provider_attempted"] += 1
            if provider.get("provider_path"):
                summary["provider_available"] += 1
            if provider.get("failure"):
                summary["failed"] += 1
            if provider.get("acquisition_duration_ms") is not None:
                summary["acquisition_duration_ms"] += provider["acquisition_duration_ms"]
                acquisition_count += 1
            if provider.get("llm_duration_ms"):
                summary["llm_duration_ms"] += provider["llm_duration_ms"]
                summary["acquisition_llm_calls"] += 1
                llm_count += 1
            for candidate in provider.get("candidates") or []:
                summary["candidate_count"] += 1
                probe = candidate.get("probe") or {}
                if probe.get("passed"):
                    summary["probe_pass_count"] += 1
                elif probe.get("status") == "failed":
                    summary["probe_fail_count"] += 1
                if (candidate.get("promotion") or {}).get("binding_id"):
                    summary["promoted_bindings"] += 1
                for reuse in candidate.get("reuse") or []:
                    summary["held_out_reuses"] += 1
                    summary["reuse_duration_ms"] += reuse.get("duration_ms", 0)
                    reuse_count += 1
                    if (reuse.get("oracle") or {}).get("passed"):
                        summary["oracle_pass_count"] += 1
                    else:
                        summary["oracle_fail_count"] += 1
                        summary["failed"] += 1
    summary["avg_acquisition_ms"] = average(summary["acquisition_duration_ms"], acquisition_count)
    summary["avg_reuse_ms"] = average(summary["reuse_duration_ms"], reuse_count)
    summary["avg_llm_ms"] = average(summary["llm_duration_ms"], llm_count)
    if summary["promoted_bindings"]:
        summary["oracle_reuse_success_rate"] = summary["oracle_pass_count"] / summary["held_out_reuses"] if summary["held_out_reuses"] else 0
    return summary


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
    summary = artifact.get("summary") or {}
    rows = []
    for task in artifact.get("tasks") or []:
        for provider in task.get("providers") or []:
            candidates = provider.get("candidates") or []
            if not candidates:
                rows.append(provider_row(task, provider, {}))
            for candidate in candidates:
                rows.append(provider_row(task, provider, candidate))
    cards = "".join(f"<div class='card'>{escape(key)}<br><strong>{escape(value)}</strong></div>" for key, value in summary.items())
    return f"""<!doctype html>
<html>
<head>
<meta charset="utf-8">
<title>CAL CLI v0 Benchmark</title>
<style>
body{{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;margin:24px;color:#222}}
.grid{{display:grid;grid-template-columns:repeat(auto-fit,minmax(160px,1fr));gap:12px;margin:16px 0}}
.card{{border:1px solid #ddd;border-radius:8px;padding:12px;background:#fafafa}}
table{{border-collapse:collapse;width:100%;font-size:14px}}
td,th{{border:1px solid #ddd;padding:8px;vertical-align:top}}
th{{background:#f5f5f5;text-align:left}}
code{{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:12px}}
.ok{{color:#126b31;font-weight:600}}
.fail{{color:#9b1c1c;font-weight:600}}
.muted{{color:#666}}
</style>
</head>
<body>
<h1>CAL CLI v0 Benchmark</h1>
<pre>{escape(json.dumps({k: artifact.get(k) for k in ["run_id", "mode", "status", "selected_tasks", "llm"]}, indent=2))}</pre>
<div class="grid">{cards}</div>
<h2>Cases</h2>
<table>
<tr><th>Task</th><th>Provider</th><th>Capability</th><th>Probe</th><th>Promotion</th><th>Held-out reuse</th><th>Oracle</th><th>Failure</th></tr>
{''.join(rows)}
</table>
<h2>Capability model</h2>
<pre>{escape(json.dumps(artifact.get("capability_model", {}), indent=2))}</pre>
<h2>Failure taxonomy</h2>
<pre>{escape(json.dumps(artifact.get("failure_taxonomy", []), indent=2))}</pre>
</body>
</html>
"""


def provider_row(task: dict[str, Any], provider: dict[str, Any], candidate: dict[str, Any]) -> str:
    reuse_items = candidate.get("reuse") or []
    reuse_text = "<br>".join(
        f"{escape(item.get('fixture_id'))}: {status(item.get('status'))} ({item.get('duration_ms', 0)} ms)"
        for item in reuse_items
    ) or "<span class='muted'>not run</span>"
    oracle_text = "<br>".join(
        f"{escape(item.get('fixture_id'))}: {status('passed' if (item.get('oracle') or {}).get('passed') else 'failed')}"
        for item in reuse_items
    ) or "<span class='muted'>not run</span>"
    promotion = candidate.get("promotion") or {}
    failure = provider.get("failure") or candidate_failure(candidate)
    return f"""<tr>
<td><code>{escape(task.get('id'))}</code><br>{escape(task.get('capability_goal'))}</td>
<td><code>{escape(provider.get('id'))}</code><br>{escape(provider.get('provider_path', provider.get('command', '')))}</td>
<td><code>{escape(candidate.get('capability_id', ''))}</code><br>{escape(candidate.get('description', ''))}</td>
<td>{status((candidate.get('probe') or {}).get('status'))}</td>
<td>{escape(promotion.get('capability_action', ''))}/{escape(promotion.get('binding_action', ''))}<br><code>{escape(promotion.get('binding_id', ''))}</code></td>
<td>{reuse_text}</td>
<td>{oracle_text}</td>
<td>{failure_text(failure)}</td>
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
    return None


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
    if summary.get("oracle_pass_count", 0) < 1:
        raise SystemExit("benchmark produced no oracle-passing held-out reuse")
    if mode == MODE_REPLAY:
        if summary.get("oracle_fail_count", 0) > 0:
            raise SystemExit("benchmark produced oracle failures")
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
        for candidate in provider.get("candidates") or []:
            if (candidate.get("promotion") or {}).get("binding_id"):
                count += 1
    return count


def count_oracle_passed(task: dict[str, Any]) -> int:
    count = 0
    for provider in task.get("providers") or []:
        count += count_oracle_passed_provider(provider)
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
            seen = True
            if not (reuse.get("oracle") or {}).get("passed"):
                return False
    return seen


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


def escape(value: Any) -> str:
    return html.escape(str(value if value is not None else ""))


if __name__ == "__main__":
    raise SystemExit(main())
