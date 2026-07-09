from __future__ import annotations

from typing import Any

from constants import (
    BASELINE_DIRECT_CLI,
    BASELINE_LLM_ONESHOT,
    EXPERIMENT_ACQUISITION,
    EXPERIMENT_CAPABILITY_STRUCTURE,
    EXPERIMENT_REPEATED_REUSE,
    EXPERIMENT_VERIFICATION_FAILURE,
    EXPERIMENTS,
    MODE_REPLAY,
    STATUS_SKIPPED,
    UPSTREAM_ACQUISITION_FAILED,
    UPSTREAM_NO_PROMOTED_BINDING,
)
from util import average, ratio

ACQUISITION_GATE_TARGET = 0.85
VERIFICATION_BLOCK_GATE_TARGET = 0.95
CAPABILITY_STRUCTURE_GATE_TARGET = 0.90
REUSE_GATE_TARGET = 0.90


def update_artifact_metrics(artifact: dict[str, Any], mode: str) -> None:
    cases = artifact.get("cases") or []
    artifact["summary"] = summarize(cases, mode)
    artifact["scores"] = score(artifact["summary"], mode)
    artifact["coverage"] = coverage(cases)
    artifact["capability_model"] = capability_model(cases)
    artifact["failure_taxonomy"] = failure_taxonomy(cases)
    artifact["experiment_gates"] = experiment_gates(artifact["summary"], artifact["capability_model"])


def summarize(cases: list[dict[str, Any]], mode: str = MODE_REPLAY) -> dict[str, Any]:
    experiment_summaries = {experiment: summarize_experiment(cases, experiment, mode) for experiment in EXPERIMENTS}
    total = new_summary()
    for case in cases:
        merge_summary(total, summarize_case(case, mode))
    return {
        "total": finalize_summary(total),
        "experiments": {experiment: finalize_summary(value) for experiment, value in experiment_summaries.items()},
        "baselines": summarize_baselines(cases),
    }


def summarize_experiment(cases: list[dict[str, Any]], experiment: str, mode: str) -> dict[str, Any]:
    summary = new_summary()
    for case in cases:
        if experiment not in case.get("paper_experiments", []):
            continue
        case_summary = summarize_case(case, mode)
        if experiment == EXPERIMENT_REPEATED_REUSE and not case_has_promoted_binding(case):
            case_summary["upstream_acquisition_failure_count"] += 1
            case_summary.setdefault("upstream_acquisition_failed_cases", []).append(
                {"case_id": case.get("id", ""), "case_key": case.get("case_key", ""), "reason": UPSTREAM_NO_PROMOTED_BINDING}
            )
        merge_summary(summary, case_summary)
    return summary


def summarize_case(case: dict[str, Any], mode: str) -> dict[str, Any]:
    summary = new_summary()
    summary["case_attempted"] = 1
    summary["planned_reuse_rounds"] = len(case.get("reuse_rounds") or (case.get("reuse") or {}).get("rounds") or [])
    acquisition_count = reuse_count = use_count = llm_count = 0
    for provider in case.get("providers") or []:
        summary["provider_attempted"] += 1
        if provider.get("provider_path"):
            summary["provider_available"] += 1
        if provider.get("failure"):
            summary["provider_failures"] += 1
            if mode == MODE_REPLAY and not provider.get("optional"):
                summary["failed"] += 1
        if provider.get("acquisition_duration_ms") is not None:
            summary["acquisition_duration_ms"] += provider["acquisition_duration_ms"]
            acquisition_count += 1
        if provider.get("llm_duration_ms"):
            summary["llm_duration_ms"] += provider["llm_duration_ms"]
        llm_calls = provider.get("llm_call_count", 0)
        if llm_calls:
            summary["acquisition_llm_calls"] += llm_calls
            llm_count += llm_calls
        summary["prompt_tokens"] += provider.get("prompt_tokens", 0)
        summary["completion_tokens"] += provider.get("completion_tokens", 0)
        summary["total_tokens"] += provider.get("total_tokens", 0)
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
    promoted = case_has_promoted_binding(case)
    for use in case.get("use") or []:
        summary["held_out_uses"] += 1
        if promoted:
            summary["eligible_held_out_uses"] += 1
        summary["use_duration_ms"] += use.get("duration_ms", 0)
        use_count += 1
        if ((use.get("selection") or {}).get("source")) == "llm":
            summary["run_stage_llm_calls"] += 1
        if (use.get("oracle") or {}).get("passed"):
            summary["use_oracle_pass_count"] += 1
            if promoted:
                summary["conditional_use_oracle_pass_count"] += 1
        else:
            summary["use_oracle_fail_count"] += 1
            summary["intent_use_failures"] += 1
            summary["failed"] += 1
    summary["_acquisition_count"] = acquisition_count
    summary["_reuse_count"] = reuse_count
    summary["_use_count"] = use_count
    summary["_llm_count"] = llm_count
    return summary


def new_summary() -> dict[str, Any]:
    return {
        "case_attempted": 0,
        "planned_reuse_rounds": 0,
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
        "eligible_held_out_uses": 0,
        "oracle_pass_count": 0,
        "oracle_fail_count": 0,
        "use_oracle_pass_count": 0,
        "conditional_use_oracle_pass_count": 0,
        "use_oracle_fail_count": 0,
        "failed": 0,
        "provider_failures": 0,
        "candidate_negative_evidence": 0,
        "direct_reuse_failures": 0,
        "intent_use_failures": 0,
        "upstream_acquisition_failure_count": 0,
        "upstream_acquisition_failed_cases": [],
        "acquisition_duration_ms": 0,
        "reuse_duration_ms": 0,
        "use_duration_ms": 0,
        "llm_duration_ms": 0,
        "acquisition_llm_calls": 0,
        "run_stage_llm_calls": 0,
        "prompt_tokens": 0,
        "completion_tokens": 0,
        "total_tokens": 0,
        "_acquisition_count": 0,
        "_reuse_count": 0,
        "_use_count": 0,
        "_llm_count": 0,
    }


def merge_summary(target: dict[str, Any], source: dict[str, Any]) -> None:
    for key, value in source.items():
        if isinstance(value, int):
            target[key] = target.get(key, 0) + value
        elif key == "upstream_acquisition_failed_cases" and value:
            target.setdefault(key, []).extend(value)


def finalize_summary(summary: dict[str, Any]) -> dict[str, Any]:
    result = {key: value for key, value in summary.items() if not key.startswith("_")}
    result["avg_acquisition_ms"] = average(summary["acquisition_duration_ms"], summary["_acquisition_count"])
    result["avg_reuse_ms"] = average(summary["reuse_duration_ms"], summary["_reuse_count"])
    result["avg_use_ms"] = average(summary["use_duration_ms"], summary["_use_count"])
    result["avg_llm_ms"] = average(summary["llm_duration_ms"], summary["_llm_count"])
    result["oracle_reuse_success_rate"] = ratio(summary["oracle_pass_count"], summary["held_out_reuses"])
    result["oracle_use_success_rate"] = ratio(summary["use_oracle_pass_count"], summary["held_out_uses"])
    result["conditional_oracle_use_success_rate"] = ratio(summary["conditional_use_oracle_pass_count"], summary["eligible_held_out_uses"])
    return result


def summarize_baselines(cases: list[dict[str, Any]]) -> dict[str, Any]:
    summaries = {
        BASELINE_DIRECT_CLI: new_baseline_summary(),
        BASELINE_LLM_ONESHOT: new_baseline_summary(),
    }
    for case in cases:
        if EXPERIMENT_REPEATED_REUSE not in case.get("paper_experiments", []):
            continue
        for baseline, results in (case.get("baselines") or {}).items():
            summary = summaries.setdefault(baseline, new_baseline_summary())
            for result in results or []:
                merge_baseline_result(summary, result)
    return {name: finalize_baseline_summary(value) for name, value in summaries.items()}


def new_baseline_summary() -> dict[str, int]:
    return {
        "attempted": 0,
        "passed": 0,
        "failed": 0,
        "duration_ms": 0,
        "llm_calls": 0,
        "prompt_tokens": 0,
        "completion_tokens": 0,
        "total_tokens": 0,
    }


def merge_baseline_result(summary: dict[str, int], result: dict[str, Any]) -> None:
    if result.get("status") == STATUS_SKIPPED:
        return
    summary["attempted"] += 1
    summary["duration_ms"] += result.get("duration_ms", 0)
    if (result.get("oracle") or {}).get("passed"):
        summary["passed"] += 1
    else:
        summary["failed"] += 1
    llm = result.get("llm") or {}
    usage = llm.get("usage") or {}
    if llm:
        summary["llm_calls"] += 1
    summary["prompt_tokens"] += int(usage.get("prompt_tokens") or 0)
    summary["completion_tokens"] += int(usage.get("completion_tokens") or 0)
    summary["total_tokens"] += int(usage.get("total_tokens") or 0)


def finalize_baseline_summary(summary: dict[str, int]) -> dict[str, Any]:
    result = dict(summary)
    result["success_rate"] = ratio(summary["passed"], summary["attempted"])
    result["avg_duration_ms"] = average(summary["duration_ms"], summary["attempted"])
    result["avg_tokens"] = average(summary["total_tokens"], summary["llm_calls"])
    return result


def score(summary: dict[str, Any], mode: str) -> dict[str, Any]:
    total = summary.get("total") or {}
    reuse = (summary.get("experiments") or {}).get(EXPERIMENT_REPEATED_REUSE) or {}
    use_rate = ratio(reuse.get("use_oracle_pass_count", 0), reuse.get("held_out_uses", 0))
    conditional_rate = ratio(reuse.get("conditional_use_oracle_pass_count", 0), reuse.get("eligible_held_out_uses", 0))
    return {
        "profile": mode,
        "closed_loop_success_rate": use_rate,
        "intent_use_success_rate": use_rate,
        "conditional_reuse_success_rate": conditional_rate,
        "provider_yield_rate": ratio(total.get("providers_with_promoted_bindings", 0), total.get("provider_available", 0)),
        "probe_pass_rate": ratio(total.get("probe_pass_count", 0), total.get("candidate_count", 0)),
        "promotion_yield": ratio(total.get("promoted_bindings", 0), total.get("probe_pass_count", 0)),
        "negative_evidence_count": total.get("candidate_negative_evidence", 0),
        "failed_count": total.get("failed", 0),
        "provider_acquisition_total_ms": total.get("acquisition_duration_ms", 0),
        "provider_acquisition_avg_ms": total.get("avg_acquisition_ms", 0),
        "proposal_llm_total_ms": total.get("llm_duration_ms", 0),
        "proposal_llm_avg_ms": total.get("avg_llm_ms", 0),
        "proposal_llm_calls": total.get("acquisition_llm_calls", 0),
        "proposal_prompt_tokens": total.get("prompt_tokens", 0),
        "proposal_completion_tokens": total.get("completion_tokens", 0),
        "proposal_total_tokens": total.get("total_tokens", 0),
        "acquisition_local_overhead_total_ms": max(0, total.get("acquisition_duration_ms", 0) - total.get("llm_duration_ms", 0)),
    }


def coverage(cases: list[dict[str, Any]]) -> dict[str, Any]:
    case_ids = {case.get("id", "") for case in cases}
    provider_pairs = set()
    provider_ids = set()
    provider_classes = set()
    levels: dict[str, int] = {}
    domains = set()
    experiments = set()
    for case in cases:
        if case.get("level"):
            levels[case["level"]] = levels.get(case["level"], 0) + 1
        if case.get("domain"):
            domains.add(case["domain"])
        experiments.update(case.get("paper_experiments") or [])
        if case.get("provider_class"):
            provider_classes.add(case["provider_class"])
        for provider in case.get("providers") or []:
            provider_id = provider.get("id", "")
            if provider_id:
                provider_ids.add(provider_id)
                provider_pairs.add((case.get("id", ""), provider_id))
            if provider.get("provider_class"):
                provider_classes.add(provider["provider_class"])
    return {
        "distinct_case_count": len(case_ids),
        "distinct_provider_case_pair_count": len(provider_pairs),
        "distinct_provider_count": len(provider_ids),
        "levels": levels,
        "domains": sorted(domains),
        "provider_classes": sorted(provider_classes),
        "paper_experiments": sorted(experiments),
    }


def capability_model(cases: list[dict[str, Any]]) -> dict[str, Any]:
    providers: dict[str, set[str]] = {}
    capabilities: dict[str, set[str]] = {}
    promoted_by_case = {case.get("id", ""): case_has_promoted_binding(case) for case in cases}
    for case in cases:
        for provider in case.get("providers") or []:
            provider_name = provider.get("id") or provider.get("provider_id") or ""
            for candidate in provider.get("candidates") or []:
                promotion = candidate.get("promotion") or {}
                binding_id = promotion.get("binding_id", "")
                capability_id = promotion.get("capability_id") or candidate.get("capability_id", "")
                if not binding_id or not capability_id:
                    continue
                providers.setdefault(provider_name, set()).add(capability_id)
                capabilities.setdefault(capability_id, set()).add(provider_name)
    checks = capability_model_checks(cases, providers, capabilities, promoted_by_case)
    passed = sum(1 for check in checks if check.get("status") == "passed")
    failed = sum(1 for check in checks if check.get("status") == "failed")
    skipped = sum(1 for check in checks if check.get("status") == "skipped")
    return {
        "providers": {provider: sorted(caps) for provider, caps in sorted(providers.items())},
        "capabilities": {capability: sorted(items) for capability, items in sorted(capabilities.items())},
        "multi_capability_providers": sum(1 for caps in providers.values() if len(caps) > 1),
        "multi_binding_capabilities": sum(1 for provider_names in capabilities.values() if len(provider_names) > 1),
        "checks": checks,
        "check_passed": passed,
        "check_failed": failed,
        "check_skipped": skipped,
    }


def capability_model_checks(
    cases: list[dict[str, Any]],
    providers: dict[str, set[str]],
    capabilities: dict[str, set[str]],
    promoted_by_case: dict[str, bool],
) -> list[dict[str, Any]]:
    checks: list[dict[str, Any]] = []
    for case in cases:
        if EXPERIMENT_CAPABILITY_STRUCTURE not in case.get("paper_experiments", []):
            continue
        expectations = case.get("capability_layer_checks") or {}
        if not expectations:
            continue
        if not promoted_by_case.get(case.get("id", ""), False):
            checks.append({"case_id": case.get("id", ""), "expectation": "promoted_binding", "status": "skipped", "skip_reason": UPSTREAM_ACQUISITION_FAILED})
            continue
        handled_provider_count = False
        if expectations.get("expect_multi_binding"):
            expected = int(expectations.get("expected_provider_count") or 2)
            actual = max((len(provider_names) for provider_names in capabilities.values()), default=0)
            checks.append(check_result(case, "multi_binding", actual, expected))
            handled_provider_count = True
        if expectations.get("expect_provider_multi_capability_with"):
            actual = max((len(caps) for caps in providers.values()), default=0)
            checks.append(check_result(case, "provider_multi_capability", actual, 2))
        if expectations.get("expected_provider_count") and not handled_provider_count:
            expected = int(expectations["expected_provider_count"])
            actual = max((len(provider_names) for provider_names in capabilities.values()), default=0)
            checks.append(check_result(case, "expected_provider_count", actual, expected))
    return checks


def check_result(case: dict[str, Any], expectation: str, actual: int, expected: int) -> dict[str, Any]:
    return {
        "case_id": case.get("id", ""),
        "case_key": case.get("case_key", ""),
        "expectation": expectation,
        "actual": actual,
        "expected": expected,
        "status": "passed" if actual >= expected else "failed",
    }


def failure_taxonomy(cases: list[dict[str, Any]]) -> list[dict[str, Any]]:
    counts: dict[tuple[str, str], int] = {}
    for case in cases:
        collect_failures(counts, case.get("providers") or [])
        collect_failures(counts, case.get("use") or [])
        for results in (case.get("baselines") or {}).values():
            collect_failures(counts, results or [])
        if EXPERIMENT_REPEATED_REUSE in case.get("paper_experiments", []) and not case_has_promoted_binding(case):
            increment_failure(counts, UPSTREAM_ACQUISITION_FAILED, UPSTREAM_NO_PROMOTED_BINDING)
    return [{"stage": stage, "code": code, "count": count} for (stage, code), count in sorted(counts.items())]


def collect_failures(counts: dict[tuple[str, str], int], records: list[dict[str, Any]]) -> None:
    for record in records:
        failure = record.get("failure") or {}
        if failure:
            increment_failure(counts, failure.get("stage", ""), failure.get("code", ""))
        for candidate in record.get("candidates") or []:
            probe_failure = ((candidate.get("probe") or {}).get("error") or {})
            if probe_failure:
                increment_failure(counts, probe_failure.get("stage", ""), probe_failure.get("code", ""))


def increment_failure(counts: dict[tuple[str, str], int], stage: str, code: str) -> None:
    counts[(stage or "unknown", code or "unknown")] = counts.get((stage or "unknown", code or "unknown"), 0) + 1


def experiment_gates(summary: dict[str, Any], model: dict[str, Any]) -> dict[str, Any]:
    experiments = summary.get("experiments") or {}
    acquisition = experiments.get(EXPERIMENT_ACQUISITION) or {}
    verification = experiments.get(EXPERIMENT_VERIFICATION_FAILURE) or {}
    reuse = experiments.get(EXPERIMENT_REPEATED_REUSE) or {}
    invalid_cases = verification.get("case_attempted", 0)
    blocked_invalid = max(0, invalid_cases - verification.get("providers_with_promoted_bindings", 0))
    structure_denominator = model.get("check_passed", 0) + model.get("check_failed", 0)
    return {
        EXPERIMENT_ACQUISITION: gate(
            "promoted_provider_case_yield",
            acquisition.get("providers_with_promoted_bindings", 0),
            acquisition.get("provider_available", 0),
            ACQUISITION_GATE_TARGET,
        ),
        EXPERIMENT_VERIFICATION_FAILURE: {
            **gate("blocked_invalid_candidate_rate", blocked_invalid, invalid_cases, VERIFICATION_BLOCK_GATE_TARGET),
            "false_promotions": verification.get("providers_with_promoted_bindings", 0),
            "passed": verification.get("providers_with_promoted_bindings", 0) == 0 and ratio(blocked_invalid, invalid_cases) >= VERIFICATION_BLOCK_GATE_TARGET,
        },
        EXPERIMENT_CAPABILITY_STRUCTURE: {
            **gate("structure_check_pass_rate", model.get("check_passed", 0), structure_denominator, CAPABILITY_STRUCTURE_GATE_TARGET),
            "skipped": model.get("check_skipped", 0),
        },
        EXPERIMENT_REPEATED_REUSE: gate(
            "held_out_use_oracle_pass_rate",
            reuse.get("use_oracle_pass_count", 0),
            reuse.get("held_out_uses", 0),
            REUSE_GATE_TARGET,
        ),
    }


def gate(metric: str, numerator: int, denominator: int, target: float) -> dict[str, Any]:
    actual = ratio(numerator, denominator)
    return {
        "metric": metric,
        "numerator": numerator,
        "denominator": denominator,
        "actual": actual,
        "target": target,
        "passed": actual >= target if denominator else False,
    }


def case_has_promoted_binding(case: dict[str, Any]) -> bool:
    return any(provider_has_promoted_binding(provider) for provider in case.get("providers") or [])


def provider_has_promoted_binding(provider: dict[str, Any]) -> bool:
    return any((candidate.get("promotion") or {}).get("binding_id") for candidate in provider.get("candidates") or [])
