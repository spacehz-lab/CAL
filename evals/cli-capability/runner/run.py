#!/usr/bin/env python3
from __future__ import annotations

import argparse
import os
import platform
import shutil
import sys
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path
from typing import Any

from acquisition import AcquisitionRunner, finalize_provider_status
from baseline import BaselineRunner
from catalog import ScenarioCatalog
from constants import (
    EXPERIMENTS,
    EXPERIMENT_ACQUISITION,
    EXPERIMENT_CAPABILITY_STRUCTURE,
    EXPERIMENT_REPEATED_REUSE,
    EXPERIMENT_VERIFICATION_FAILURE,
    MODE_LIVE_LLM,
    MODE_REPLAY,
)
from oracle import OracleRunner
from report import ArtifactWriter
from reuse import ReuseRunner, use_intent
from summary import update_artifact_metrics
from util import clean_run_part, new_run_id
from workspace import Workspace, start_cald, stop_process, wait_for_cald


def main() -> int:
    args = parse_args()
    repo = Path(__file__).resolve().parents[3]
    bench = Path(__file__).resolve().parents[1]
    catalog = ScenarioCatalog(bench)
    selected_cases = catalog.select(args.experiment, args.level, args.case, args.provider_class, args.tag, args.failure_type)
    selected_experiments = selected_experiment_names(selected_cases)
    model = os.environ.get("CAL_LLM_MODEL", "") if args.mode == MODE_LIVE_LLM else ""
    run_id = new_run_id(args.mode, model)
    out_base = Path(args.out).resolve() if args.out else repo / "evals" / "out" / "cli-capability"
    run_dir = out_base / run_id
    home = Path(args.home).resolve() if args.home else run_dir / "home"
    env = benchmark_env(os.environ.copy(), bench)
    env["CAL_HOME"] = str(home)

    calctl = resolve_executable(args.calctl)
    cald = resolve_executable(args.cald)
    worker_count = effective_jobs(args)
    if args.no_start_cald and worker_count != 1:
        raise SystemExit("--no-start-cald can only be used with --jobs 1")

    artifact = new_artifact(run_id, args, selected_cases, selected_experiments, model, worker_count)
    writer = ArtifactWriter(run_dir, artifact)
    writer.write()

    if args.no_start_cald:
        wait_for_cald(calctl, repo, env)
        workspace = Workspace(repo, home, calctl, env)
        oracle = OracleRunner(repo, bench)
        benchmark = BenchmarkRunner(bench, home, workspace, args.mode, oracle)
        baselines = BaselineRunner(repo, bench, home, oracle, args.mode, env)
        for case in selected_cases:
            result = run_case(case, benchmark, baselines)
            artifact["cases"].append(result)
            update_artifact_metrics(artifact, args.mode)
            writer.write()
        finish_artifact(artifact, args.mode, writer)
        return 0

    with ThreadPoolExecutor(max_workers=worker_count) as executor:
        futures = {
            executor.submit(run_case_shard, case, repo, bench, run_dir, calctl, cald, args.mode): case
            for case in selected_cases
        }
        for future in as_completed(futures):
            case = futures[future]
            result = future.result()
            artifact["cases"].append(result)
            artifact["cases"].sort(key=lambda item: item.get("case_key", item.get("id", "")))
            update_artifact_metrics(artifact, args.mode)
            writer.write()
            print(
                f"cli-capability: completed case={case.get('case_key', case.get('id', ''))} "
                f"experiments={','.join(case.get('paper_experiments') or [])} "
                f"providers={len(result.get('providers', []))} promoted={count_promotions(result)} "
                f"use_oracle_passed={count_use_oracle_passed(result)}",
                flush=True,
            )

    finish_artifact(artifact, args.mode, writer)
    return 0


class BenchmarkRunner:
    def __init__(self, bench: Path, home: Path, workspace: Workspace, mode: str, oracle: OracleRunner) -> None:
        self.mode = mode
        self.acquisition = AcquisitionRunner(bench, workspace, mode)
        self.reuse = ReuseRunner(bench, home, workspace, oracle)

    def run_case(self, case: dict[str, Any]) -> dict[str, Any]:
        result = self.new_case_result(case)
        if should_run_acquisition(case):
            for provider in case["providers"]:
                provider_result = self.acquisition.run_provider(case, provider)
                finalize_provider_status(provider_result, self.mode, False)
                result["providers"].append(provider_result)
        if EXPERIMENT_REPEATED_REUSE in case.get("paper_experiments", []):
            self.reuse.run_intent_uses(case, result)
        return result

    def new_case_result(self, case: dict[str, Any]) -> dict[str, Any]:
        return {
            "id": case["id"],
            "case_key": case.get("case_key", case["id"]),
            "scenario_group": case.get("scenario_group", ""),
            "paper_experiments": case.get("paper_experiments") or [],
            "scenario_tags": case.get("scenario_tags") or [],
            "provider_class": case.get("provider_class", ""),
            "acquisition_mode": case.get("acquisition_mode", ""),
            "failure_type": case.get("failure_type", ""),
            "level": case.get("level", ""),
            "domain": case.get("domain", ""),
            "intent": use_intent(case),
            "description": case.get("description", ""),
            "capability_layer_checks": case.get("capability_layer_checks") or {},
            "reuse_rounds": [round_value.get("id", "") for round_value in (case.get("reuse") or {}).get("rounds") or []],
            "providers": [],
            "use": [],
            "baselines": {},
        }


def run_case_shard(case: dict[str, Any], repo: Path, bench: Path, run_dir: Path, calctl: str, cald: str, mode: str) -> dict[str, Any]:
    shard = run_dir / "shards" / clean_run_part(case.get("case_key", case.get("id", "case")))
    home = shard / "home"
    env = benchmark_env(os.environ.copy(), bench)
    env["CAL_HOME"] = str(home)
    process = None
    try:
        process = start_cald(cald, repo, env, shard)
        wait_for_cald(calctl, repo, env)
        workspace = Workspace(repo, home, calctl, env)
        oracle = OracleRunner(repo, bench)
        benchmark = BenchmarkRunner(bench, home, workspace, mode, oracle)
        baselines = BaselineRunner(repo, bench, home, oracle, mode, env)
        result = run_case(case, benchmark, baselines)
        result["shard"] = {"path": str(shard), "home": str(home)}
        return result
    finally:
        if process is not None:
            stop_process(process)


def run_case(case: dict[str, Any], benchmark: BenchmarkRunner, baselines: BaselineRunner) -> dict[str, Any]:
    print(
        f"cli-capability: starting case={case.get('case_key', case.get('id', ''))} "
        f"experiments={','.join(case.get('paper_experiments') or [])}",
        flush=True,
    )
    result = benchmark.run_case(case)
    if EXPERIMENT_REPEATED_REUSE in case.get("paper_experiments", []):
        result["baselines"] = baselines.run_case(case)
    return result


def should_run_acquisition(case: dict[str, Any]) -> bool:
    experiments = set(case.get("paper_experiments") or [])
    return bool(
        experiments.intersection(
            {
                EXPERIMENT_ACQUISITION,
                EXPERIMENT_VERIFICATION_FAILURE,
                EXPERIMENT_CAPABILITY_STRUCTURE,
                EXPERIMENT_REPEATED_REUSE,
            }
        )
    )


def benchmark_env(env: dict[str, str], bench: Path) -> dict[str, str]:
    tools = bench / "tools" / "bin"
    if tools.exists():
        env["PATH"] = str(tools) + os.pathsep + env.get("PATH", "")
    return env


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run the CLI Capability paper benchmark.")
    parser.add_argument("--mode", choices=[MODE_REPLAY, MODE_LIVE_LLM], default=MODE_REPLAY)
    parser.add_argument("--experiment", default=",".join(EXPERIMENTS), help="comma-separated paper experiments")
    parser.add_argument("--case", default="", help="comma-separated case ids or scenario_group:case ids")
    parser.add_argument("--provider-class", default="", help="comma-separated paper provider classes")
    parser.add_argument("--tag", default="", help="comma-separated scenario tags")
    parser.add_argument("--failure-type", default="", help="comma-separated failure types")
    parser.add_argument("--level", choices=["focus", "full"], default="focus", help="select benchmark cases by level")
    parser.add_argument("--jobs", type=int, default=1, help="parallel case workers")
    parser.add_argument("--llm-jobs", type=int, default=2, help="parallel live LLM case workers")
    parser.add_argument("--calctl", default="calctl")
    parser.add_argument("--cald", default="cald")
    parser.add_argument("--home", default="")
    parser.add_argument("--out", default="")
    parser.add_argument("--no-start-cald", action="store_true")
    return parser.parse_args()


def new_artifact(
    run_id: str,
    args: argparse.Namespace,
    cases: list[dict[str, Any]],
    experiments: list[str],
    model: str,
    jobs: int,
) -> dict[str, Any]:
    return {
        "run": {
            "id": run_id,
            "mode": args.mode,
            "status": "running",
            "level": args.level,
            "selected_experiments": experiments,
            "selected_cases": [case.get("case_key", case["id"]) for case in cases],
            "jobs": jobs,
            "goos": sys.platform,
            "goarch": platform.machine(),
            "llm": {
                "api": os.environ.get("CAL_LLM_API", "") if args.mode == MODE_LIVE_LLM else "",
                "model": model,
                "base_url_configured": bool(os.environ.get("CAL_LLM_BASE_URL")) if args.mode == MODE_LIVE_LLM else False,
            },
        },
        "source": "cli-capability paper benchmark",
        "status": "running",
        "cases": [],
        "summary": {},
        "scores": {},
        "coverage": {},
        "capability_model": {},
        "failure_taxonomy": [],
        "experiment_gates": {},
    }


def finish_artifact(artifact: dict[str, Any], mode: str, writer: ArtifactWriter) -> None:
    update_artifact_metrics(artifact, mode)
    artifact["status"] = "completed"
    artifact["run"]["status"] = "completed"
    writer.write()
    print(f"summary: {writer.summary_path}")
    print(f"flow: {writer.flow_path}")
    print(f"html: {writer.html_path}")
    print(f"artifact: {writer.artifact_path}")


def effective_jobs(args: argparse.Namespace) -> int:
    jobs = max(1, int(args.jobs or 1))
    if args.mode == MODE_LIVE_LLM:
        return max(1, min(jobs, int(args.llm_jobs or 1)))
    return jobs


def selected_experiment_names(cases: list[dict[str, Any]]) -> list[str]:
    selected = set()
    for case in cases:
        selected.update(case.get("paper_experiments") or [])
    return [experiment for experiment in EXPERIMENTS if experiment in selected]


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
