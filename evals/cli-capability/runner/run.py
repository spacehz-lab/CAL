#!/usr/bin/env python3
from __future__ import annotations

import argparse
import os
import platform
import shutil
import sys
from pathlib import Path
from typing import Any

from acquisition import AcquisitionRunner, finalize_provider_status
from baseline import BaselineRunner
from catalog import SuiteCatalog
from constants import MODE_LIVE_LLM, MODE_REPLAY, SUITE_REUSE, SUITES
from oracle import OracleRunner
from report import ArtifactWriter
from reuse import ReuseRunner, use_intent
from summary import update_artifact_metrics
from util import new_run_id
from workspace import Workspace, start_cald, stop_process, wait_for_cald


def main() -> int:
    args = parse_args()
    repo = Path(__file__).resolve().parents[3]
    bench = Path(__file__).resolve().parents[1]
    catalog = SuiteCatalog(bench)
    selected_cases = catalog.select(args.suite, args.level, args.case)
    selected_suites = selected_suite_names(selected_cases)
    model = os.environ.get("CAL_LLM_MODEL", "") if args.mode == MODE_LIVE_LLM else ""
    run_id = new_run_id(args.mode, model)
    out_base = Path(args.out).resolve() if args.out else repo / "evals" / "out" / "cli-capability"
    run_dir = out_base / run_id
    home = Path(args.home).resolve() if args.home else run_dir / "home"
    env = os.environ.copy()
    env["CAL_HOME"] = str(home)

    artifact = new_artifact(run_id, args, selected_cases, selected_suites, model)
    writer = ArtifactWriter(run_dir, artifact)
    calctl = resolve_executable(args.calctl)
    cald = resolve_executable(args.cald)
    process = None
    try:
        if not args.no_start_cald:
            process = start_cald(cald, repo, env, run_dir)
        wait_for_cald(calctl, repo, env)
        workspace = Workspace(repo, home, calctl, env)
        oracle = OracleRunner(repo, bench)
        benchmark = BenchmarkRunner(bench, home, workspace, args.mode, oracle)
        baselines = BaselineRunner(repo, bench, home, oracle)
        for case in selected_cases:
            print(f"cli-capability: starting suite={case['suite']} case={case['id']} mode={args.mode}", flush=True)
            result = benchmark.run_case(case)
            if case["suite"] == SUITE_REUSE:
                result["baselines"] = baselines.run_case(case)
            artifact["suites"][case["suite"]]["cases"].append(result)
            update_artifact_metrics(artifact, args.mode)
            writer.write()
            print(
                f"cli-capability: completed suite={case['suite']} case={case['id']} "
                f"providers={len(result.get('providers', []))} "
                f"promoted={count_promotions(result)} "
                f"use_oracle_passed={count_use_oracle_passed(result)}",
                flush=True,
            )
        update_artifact_metrics(artifact, args.mode)
        artifact["status"] = "completed"
        artifact["run"]["status"] = "completed"
        writer.write()
        validate_result(args.mode, artifact)
        print(f"summary: {writer.summary_path}")
        print(f"flow: {writer.flow_path}")
        print(f"html: {writer.html_path}")
        print(f"artifact: {writer.artifact_path}")
        return 0
    finally:
        if process is not None:
            stop_process(process)


class BenchmarkRunner:
    def __init__(self, bench: Path, home: Path, workspace: Workspace, mode: str, oracle: OracleRunner) -> None:
        self.mode = mode
        self.acquisition = AcquisitionRunner(bench, workspace, mode)
        self.reuse = ReuseRunner(bench, home, workspace, oracle)

    def run_case(self, case: dict[str, Any]) -> dict[str, Any]:
        result: dict[str, Any] = {
            "id": case["id"],
            "suite": case["suite"],
            "level": case.get("level", ""),
            "domain": case.get("domain", ""),
            "intent": use_intent(case),
            "description": case.get("description", ""),
            "providers": [],
            "use": [],
            "baselines": {},
        }
        for provider in case["providers"]:
            provider_result = self.acquisition.run_provider(case, provider)
            if self.mode == MODE_REPLAY and case["suite"] == SUITE_REUSE:
                self.reuse.run_direct_reuse(case, provider_result)
            finalize_provider_status(provider_result, self.mode, case["suite"] == SUITE_REUSE)
            result["providers"].append(provider_result)
        if case["suite"] == SUITE_REUSE:
            self.reuse.run_intent_uses(case, result)
        return result


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run the CLI Capability benchmark.")
    parser.add_argument("--mode", choices=[MODE_REPLAY, MODE_LIVE_LLM], default=MODE_REPLAY)
    parser.add_argument("--suite", default=",".join(SUITES), help="comma-separated suites: acquisition,capability_model,reuse")
    parser.add_argument("--case", default="", help="comma-separated suite case ids")
    parser.add_argument("--level", choices=["focus", "full"], default="focus", help="select benchmark cases by level")
    parser.add_argument("--calctl", default="calctl")
    parser.add_argument("--cald", default="cald")
    parser.add_argument("--home", default="")
    parser.add_argument("--out", default="")
    parser.add_argument("--no-start-cald", action="store_true")
    return parser.parse_args()


def new_artifact(run_id: str, args: argparse.Namespace, cases: list[dict[str, Any]], suites: list[str], model: str) -> dict[str, Any]:
    return {
        "run": {
            "id": run_id,
            "mode": args.mode,
            "status": "running",
            "level": args.level,
            "selected_suites": suites,
            "selected_cases": [f"{case['suite']}:{case['id']}" for case in cases],
            "goos": sys.platform,
            "goarch": platform.machine(),
            "llm": {
                "api": os.environ.get("CAL_LLM_API", "") if args.mode == MODE_LIVE_LLM else "",
                "model": model,
                "base_url_configured": bool(os.environ.get("CAL_LLM_BASE_URL")) if args.mode == MODE_LIVE_LLM else False,
            },
        },
        "source": "cli-capability benchmark",
        "status": "running",
        "suites": {suite: {"cases": []} for suite in SUITES},
        "summary": {},
        "scores": {},
        "capability_model": {},
        "failure_taxonomy": [],
    }


def validate_result(mode: str, artifact: dict[str, Any]) -> None:
    reuse = (((artifact.get("summary") or {}).get("suites") or {}).get(SUITE_REUSE) or {})
    reuse_selected = SUITE_REUSE in ((artifact.get("run") or {}).get("selected_suites") or [])
    if not reuse_selected:
        return
    if reuse.get("use_oracle_pass_count", 0) < 1:
        raise SystemExit("benchmark produced no oracle-passing held-out use")
    if mode == MODE_REPLAY:
        if reuse.get("oracle_pass_count", 0) < 1:
            raise SystemExit("benchmark produced no oracle-passing direct reuse")
        if reuse.get("oracle_fail_count", 0) > 0:
            raise SystemExit("benchmark produced oracle failures")
        if reuse.get("use_oracle_fail_count", 0) > 0:
            raise SystemExit("benchmark produced use oracle failures")
        if reuse.get("failed", 0) > 0:
            raise SystemExit("replay benchmark produced failures")


def selected_suite_names(cases: list[dict[str, Any]]) -> list[str]:
    return sorted({case["suite"] for case in cases})


def count_promotions(case: dict[str, Any]) -> int:
    return sum(count_provider_promotions(provider) for provider in case.get("providers") or [])


def count_provider_promotions(provider: dict[str, Any]) -> int:
    return sum(1 for candidate in provider.get("candidates") or [] if (candidate.get("promotion") or {}).get("binding_id"))


def count_use_oracle_passed(case: dict[str, Any]) -> int:
    return sum(1 for use in case.get("use") or [] if (use.get("oracle") or {}).get("passed"))


def resolve_executable(value: str) -> str:
    path = Path(value)
    if path.parent != Path(".") or path.is_absolute():
        return str(path.resolve())
    return shutil.which(value) or value


if __name__ == "__main__":
    raise SystemExit(main())
