#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import subprocess
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from report import render_html
from util import format_duration_value, ratio, write_json


PUBLIC_SCHEMA_VERSION = "cli-capability-public-result-v1"
RESULTS_ROOT = Path("evals/results/cli-capability")
SENSITIVE_PATTERNS = [
    "/Users/",
    "sk-",
    "CAL_LLM_API_KEY",
    "api_key",
    "raw_response",
    "\"provider_path\"",
    "\"trace_id\"",
    "trace_",
    "\"shard\"",
]


def main() -> int:
    args = parse_args()
    run_dir = Path(args.run).resolve()
    output_dir = (Path(args.output).resolve() if args.output else Path.cwd() / RESULTS_ROOT / args.name)
    export_run(run_dir, output_dir)
    print(f"exported public result: {output_dir}")
    return 0


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Export a sanitized CLI capability result artifact.")
    parser.add_argument("--run", required=True, help="Path to a local evals/out run directory.")
    parser.add_argument("--name", required=True, help="Commit-ready result directory name under evals/results/cli-capability.")
    parser.add_argument("--output", default="", help="Optional explicit output directory.")
    return parser.parse_args()


def export_run(run_dir: Path, output_dir: Path) -> None:
    raw = read_json(run_dir / "summary.json")
    public = build_public_artifact(raw)
    metrics = build_metrics(public)
    provenance = build_provenance(raw, run_dir)
    output_dir.mkdir(parents=True, exist_ok=True)
    write_json(output_dir / "artifact.public.json", public)
    write_json(output_dir / "metrics.json", metrics)
    write_json(output_dir / "provenance.json", provenance)
    (output_dir / "README.md").write_text(render_readme(public, metrics, provenance), encoding="utf-8")
    (output_dir / "report.html").write_text(render_html(public), encoding="utf-8")
    assert_public_directory(output_dir)


def build_public_artifact(raw: dict[str, Any]) -> dict[str, Any]:
    return {
        "schema_version": PUBLIC_SCHEMA_VERSION,
        "run": public_run(raw.get("run") or {}),
        "status": raw.get("status", ""),
        "cases": [public_case(case) for case in raw.get("cases") or []],
        "summary": raw.get("summary") or {},
        "scores": raw.get("scores") or {},
        "coverage": raw.get("coverage") or {},
        "capability_model": raw.get("capability_model") or {},
        "discovery_coverage": raw.get("discovery_coverage") or {},
        "failure_taxonomy": raw.get("failure_taxonomy") or [],
        "experiment_gates": raw.get("experiment_gates") or {},
    }


def public_run(run: dict[str, Any]) -> dict[str, Any]:
    llm = run.get("llm") or {}
    return {
        "id": run.get("id", ""),
        "mode": run.get("mode", ""),
        "status": run.get("status", ""),
        "level": run.get("level", ""),
        "selected_experiments": run.get("selected_experiments") or [],
        "selected_cases": run.get("selected_cases") or [],
        "jobs": run.get("jobs", 0),
        "reuse_seed": run.get("reuse_seed", ""),
        "goos": run.get("goos", ""),
        "goarch": run.get("goarch", ""),
        "llm": {
            "api": llm.get("api", ""),
            "model": llm.get("model", ""),
            "base_url_configured": bool(llm.get("base_url_configured")),
        },
    }


def public_case(case: dict[str, Any]) -> dict[str, Any]:
    return {
        "id": case.get("id", ""),
        "case_key": case.get("case_key", ""),
        "scenario_group": case.get("scenario_group", ""),
        "paper_experiments": case.get("paper_experiments") or [],
        "scenario_tags": case.get("scenario_tags") or [],
        "provider_class": case.get("provider_class", ""),
        "acquisition_mode": case.get("acquisition_mode", ""),
        "failure_type": case.get("failure_type", ""),
        "level": case.get("level", ""),
        "domain": case.get("domain", ""),
        "intent": case.get("intent", ""),
        "description": case.get("description", ""),
        "capability_layer_checks": case.get("capability_layer_checks") or {},
        "expected_capabilities": case.get("expected_capabilities") or [],
        "reuse_rounds": case.get("reuse_rounds") or [],
        "providers": [public_provider(provider) for provider in case.get("providers") or []],
        "use": public_uses(case.get("use") or []),
        "baselines": public_baselines(case.get("baselines") or {}),
    }


def public_provider(provider: dict[str, Any]) -> dict[str, Any]:
    candidates = [public_candidate(candidate) for candidate in provider.get("candidates") or []]
    return {
        "id": provider.get("id", ""),
        "command": provider.get("command", ""),
        "provider_class": provider.get("provider_class", ""),
        "domains": provider.get("domains") or [],
        "optional": bool(provider.get("optional", False)),
        "status": provider.get("status", ""),
        "provider_id": provider.get("provider_id", ""),
        "provider_register_duration_ms": provider.get("provider_register_duration_ms", 0),
        "acquisition_duration_ms": provider.get("acquisition_duration_ms", 0),
        "proposal_duration_ms": provider.get("proposal_duration_ms", 0),
        "llm_duration_ms": provider.get("llm_duration_ms", 0),
        "llm_call_count": provider.get("llm_call_count", 0),
        "prompt_tokens": provider.get("prompt_tokens", 0),
        "completion_tokens": provider.get("completion_tokens", 0),
        "total_tokens": provider.get("total_tokens", 0),
        "observation_sources": provider.get("observation_sources") or [],
        "failure": public_failure(provider.get("failure") or {}),
        "candidates": candidates,
        "candidate_count": len(candidates),
        "probe_pass_count": sum(1 for item in candidates if (item.get("probe") or {}).get("passed")),
        "probe_fail_count": sum(1 for item in candidates if (item.get("probe") or {}).get("status") == "failed"),
        "promoted_bindings": sum(1 for item in candidates if (item.get("promotion") or {}).get("binding_id")),
    }


def public_candidate(candidate: dict[str, Any]) -> dict[str, Any]:
    probe = candidate.get("probe") or {}
    verification = candidate.get("verification") or {}
    promotion = candidate.get("promotion") or {}
    return {
        "capability_id": promotion.get("capability_id") or candidate.get("capability_id", ""),
        "probe": {
            "status": probe.get("status", ""),
            "passed": bool(probe.get("passed")),
            "error": public_failure(probe.get("error") or {}),
        },
        "verification": {
            "level": verification.get("level", ""),
            "method": verification.get("method", ""),
            "evidence_count": verification.get("evidence_count", 0),
            "check_count": len(verification.get("checks") or []),
        },
        "promotion": {
            "capability_action": promotion.get("capability_action", ""),
            "binding_action": promotion.get("binding_action", ""),
            "capability_id": promotion.get("capability_id", ""),
            "binding_id": promotion.get("binding_id", ""),
        },
    }


def public_uses(uses: list[dict[str, Any]]) -> list[dict[str, Any]]:
    rows = []
    for item in uses:
        rows.append(
            {
                "fixture_id": item.get("fixture_id", ""),
                "status": item.get("status", ""),
                "duration_ms": item.get("duration_ms", 0),
                "selection": {
                    "source": (item.get("selection") or {}).get("source", ""),
                    "binding_id": (item.get("selection") or {}).get("binding_id", ""),
                },
                "oracle": {"passed": bool((item.get("oracle") or {}).get("passed"))},
                "failure": public_failure(item.get("failure") or {}),
            }
        )
    return rows


def public_baselines(baselines: dict[str, Any]) -> dict[str, Any]:
    result: dict[str, Any] = {}
    for name, rows in baselines.items():
        result[name] = [
            {
                "status": row.get("status", ""),
                "duration_ms": row.get("duration_ms", 0),
                "oracle": {"passed": bool((row.get("oracle") or {}).get("passed"))},
                "failure": public_failure(row.get("failure") or {}),
            }
            for row in rows or []
        ]
    return result


def public_failure(failure: dict[str, Any]) -> dict[str, str]:
    if not failure:
        return {}
    return {
        "stage": str(failure.get("stage", "")),
        "code": str(failure.get("code", "")),
        "message": str(failure.get("message", ""))[:240],
    }


def build_metrics(public: dict[str, Any]) -> dict[str, Any]:
    summary = public.get("summary") or {}
    modes = summary.get("acquisition_modes") or {}
    gate = (public.get("experiment_gates") or {}).get("acquisition") or {}
    discovery = public.get("discovery_coverage") or {}
    return {
        "schema_version": PUBLIC_SCHEMA_VERSION,
        "run_id": (public.get("run") or {}).get("id", ""),
        "model": ((public.get("run") or {}).get("llm") or {}).get("model", ""),
        "experiment": "acquisition",
        "level": (public.get("run") or {}).get("level", ""),
        "case_count": len(public.get("cases") or []),
        "provider_count": (summary.get("experiments") or {}).get("acquisition", {}).get("provider_attempted", 0),
        "acquisition_gate": compact_gate(gate),
        "intent_guided": compact_mode(modes.get("intent_guided") or {}),
        "full_acquisition": compact_mode(modes.get("full_acquisition") or {}),
        "discovery_coverage": {
            "expected_capabilities": discovery.get("expected_capabilities", 0),
            "promoted_expected_capabilities": discovery.get("promoted_expected_capabilities", 0),
            "capability_coverage_rate": discovery.get("capability_coverage_rate", 0),
            "multi_cap_design_cases": discovery.get("multi_cap_design_cases", 0),
            "multi_cap_promoted_cases": discovery.get("multi_cap_promoted_cases", 0),
            "multi_cap_promoted_rate": discovery.get("multi_cap_promoted_rate", 0),
            "missing_expected_capabilities": discovery.get("missing_expected_capabilities", 0),
        },
    }


def compact_gate(gate: dict[str, Any]) -> dict[str, Any]:
    return {
        "metric": gate.get("metric", ""),
        "numerator": gate.get("numerator", 0),
        "denominator": gate.get("denominator", 0),
        "rate": gate.get("actual", 0),
        "target": gate.get("target", 0),
        "passed": bool(gate.get("passed")),
    }


def compact_mode(mode: dict[str, Any]) -> dict[str, Any]:
    return {
        "case_attempted": mode.get("case_attempted", 0),
        "provider_attempted": mode.get("provider_attempted", 0),
        "providers_with_promoted_bindings": mode.get("providers_with_promoted_bindings", 0),
        "candidate_count": mode.get("candidate_count", 0),
        "probe_pass_count": mode.get("probe_pass_count", 0),
        "probe_fail_count": mode.get("probe_fail_count", 0),
        "promoted_bindings": mode.get("promoted_bindings", 0),
        "avg_acquisition_ms": mode.get("avg_acquisition_ms", 0),
        "llm_calls": mode.get("acquisition_llm_calls", 0),
        "total_tokens": mode.get("total_tokens", 0),
        "provider_success_rate": ratio(mode.get("providers_with_promoted_bindings", 0), mode.get("provider_attempted", 0)),
    }


def build_provenance(raw: dict[str, Any], run_dir: Path) -> dict[str, Any]:
    run = raw.get("run") or {}
    return {
        "schema_version": PUBLIC_SCHEMA_VERSION,
        "exported_at": datetime.now(timezone.utc).isoformat(),
        "source_run_id": run.get("id", ""),
        "source_run_dir": f"evals/out/cli-capability/{run.get('id', '')}",
        "source_status": raw.get("status", ""),
        "source_run_dir_present": run_dir.exists(),
        "git": git_metadata(),
        "reproduction": {
            "mode": run.get("mode", ""),
            "experiment": ",".join(run.get("selected_experiments") or []),
            "level": run.get("level", ""),
            "jobs": run.get("jobs", 0),
            "llm_jobs": run.get("jobs", 0),
            "model": (run.get("llm") or {}).get("model", ""),
            "command": (
                "python3 evals/cli-capability/runner/run.py --mode live_llm "
                "--experiment acquisition --level full --jobs 8 --llm-jobs 8 "
                "--calctl build/bin/calctl --cald build/bin/cald"
            ),
            "llm_environment": "Set required LLM provider environment variables outside the repository.",
        },
    }


def git_metadata() -> dict[str, Any]:
    commit = run_git(["rev-parse", "HEAD"])
    dirty = bool(run_git(["status", "--short"]))
    branch = run_git(["branch", "--show-current"])
    return {"commit": commit, "branch": branch, "dirty": dirty}


def render_readme(public: dict[str, Any], metrics: dict[str, Any], provenance: dict[str, Any]) -> str:
    gate = metrics["acquisition_gate"]
    intent = metrics["intent_guided"]
    full = metrics["full_acquisition"]
    discovery = metrics["discovery_coverage"]
    run = public.get("run") or {}
    missing_rows = [
        f"- `{row.get('case_key')}` missing: {', '.join(row.get('missing_surfaces') or [])}"
        for row in (public.get("discovery_coverage") or {}).get("cases") or []
        if row.get("missing_surfaces")
    ]
    missing = "\n".join(missing_rows) or "- none"
    return f"""# CLI Capability Acquisition Result

This is a sanitized, commit-ready result selected from a local live LLM run.
Raw traces, local paths, provider paths, shard directories, prompts, raw model
responses, and credentials are intentionally excluded.

## Source

- Source run id: `{provenance.get('source_run_id', '')}`
- Mode: `{run.get('mode', '')}`
- Experiment: `acquisition`
- Level: `{run.get('level', '')}`
- Model: `{((run.get('llm') or {}).get('model', ''))}`
- Jobs: `{run.get('jobs', 0)}`

## Headline Metrics

- Acquisition gate: `{gate['numerator']} / {gate['denominator']} = {gate['rate'] * 100:.2f}%`
- Intent-guided providers with promoted bindings: `{intent['providers_with_promoted_bindings']} / {intent['provider_attempted']}`
- Full-acquisition providers with promoted bindings: `{full['providers_with_promoted_bindings']} / {full['provider_attempted']}`
- Full discovery coverage: `{discovery['promoted_expected_capabilities']} / {discovery['expected_capabilities']} = {discovery['capability_coverage_rate'] * 100:.1f}%`
- Multi-cap promoted provider suites: `{discovery['multi_cap_promoted_cases']} / {discovery['multi_cap_design_cases']} = {discovery['multi_cap_promoted_rate'] * 100:.1f}%`

## Known Gaps

{missing}

## Files

- `metrics.json`: compact paper-facing metrics.
- `artifact.public.json`: sanitized machine-readable result.
- `report.html`: sanitized HTML report generated from `artifact.public.json`.
- `provenance.json`: non-secret reproduction metadata.

## Timing And Cost Signals

- Average intent-guided acquisition latency: `{format_duration_value(intent['avg_acquisition_ms'])}`
- Average full-acquisition latency: `{format_duration_value(full['avg_acquisition_ms'])}`
- Total proposal tokens: `{(public.get('scores') or {}).get('proposal_total_tokens', 0)}`
"""


def assert_public_directory(output_dir: Path) -> None:
    leaks = []
    for path in sorted(output_dir.iterdir()):
        if not path.is_file():
            continue
        text = path.read_text(encoding="utf-8")
        for pattern in SENSITIVE_PATTERNS:
            if pattern in text:
                leaks.append(f"{path.name}: {pattern}")
    if leaks:
        raise RuntimeError("public result contains sensitive fields:\n" + "\n".join(leaks))


def read_json(path: Path) -> dict[str, Any]:
    with path.open("r", encoding="utf-8") as handle:
        value = json.load(handle)
    if not isinstance(value, dict):
        raise RuntimeError(f"{path} must contain a JSON object")
    return value


def run_git(args: list[str]) -> str:
    try:
        completed = subprocess.run(["git", *args], check=False, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True)
    except OSError:
        return ""
    return completed.stdout.strip() if completed.returncode == 0 else ""


if __name__ == "__main__":
    raise SystemExit(main())
