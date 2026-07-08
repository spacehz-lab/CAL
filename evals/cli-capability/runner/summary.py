from __future__ import annotations

from typing import Any

from constants import BASELINE_DIRECT_CLI, MODE_REPLAY, STATUS_SKIPPED, SUITES
from reuse import provider_has_promoted_binding
from util import average, ratio


def update_artifact_metrics(artifact: dict[str, Any], mode: str) -> None:
    artifact["summary"] = summarize(artifact.get("suites") or {}, mode)
    artifact["scores"] = score(artifact["summary"], mode)
    artifact["capability_model"] = capability_model(artifact.get("suites") or {})
    artifact["failure_taxonomy"] = failure_taxonomy(artifact.get("suites") or {})


def summarize(suites: dict[str, Any], mode: str = MODE_REPLAY) -> dict[str, Any]:
    suite_summaries = {suite: summarize_suite((suites.get(suite) or {}).get("cases") or [], mode) for suite in SUITES}
    total = new_summary()
    for suite_summary in suite_summaries.values():
        merge_summary(total, suite_summary)
    return {
        "total": finalize_summary(total),
        "suites": {suite: finalize_summary(summary) for suite, summary in suite_summaries.items()},
        "baselines": summarize_baselines(suites),
    }


def summarize_suite(cases: list[dict[str, Any]], mode: str) -> dict[str, Any]:
    summary = new_summary()
    summary["case_attempted"] = len(cases)
    acquisition_count = reuse_count = use_count = llm_count = 0
    for case in cases:
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
        for use in case.get("use") or []:
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
    summary["_acquisition_count"] = acquisition_count
    summary["_reuse_count"] = reuse_count
    summary["_use_count"] = use_count
    summary["_llm_count"] = llm_count
    return summary


def new_summary() -> dict[str, Any]:
    return {
        "case_attempted": 0,
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
        "_acquisition_count": 0,
        "_reuse_count": 0,
        "_use_count": 0,
        "_llm_count": 0,
    }


def merge_summary(target: dict[str, Any], source: dict[str, Any]) -> None:
    for key, value in source.items():
        if isinstance(value, int):
            target[key] = target.get(key, 0) + value


def finalize_summary(summary: dict[str, Any]) -> dict[str, Any]:
    result = {key: value for key, value in summary.items() if not key.startswith("_")}
    result["avg_acquisition_ms"] = average(summary["acquisition_duration_ms"], summary["_acquisition_count"])
    result["avg_reuse_ms"] = average(summary["reuse_duration_ms"], summary["_reuse_count"])
    result["avg_use_ms"] = average(summary["use_duration_ms"], summary["_use_count"])
    result["avg_llm_ms"] = average(summary["llm_duration_ms"], summary["_llm_count"])
    result["oracle_reuse_success_rate"] = ratio(summary["oracle_pass_count"], summary["held_out_reuses"])
    result["oracle_use_success_rate"] = ratio(summary["use_oracle_pass_count"], summary["held_out_uses"])
    return result


def summarize_baselines(suites: dict[str, Any]) -> dict[str, Any]:
    direct = {"attempted": 0, "passed": 0, "failed": 0, "duration_ms": 0}
    for suite_data in suites.values():
        for case in suite_data.get("cases") or []:
            for result in ((case.get("baselines") or {}).get(BASELINE_DIRECT_CLI) or []):
                direct["attempted"] += 1
                direct["duration_ms"] += result.get("duration_ms", 0)
                if (result.get("oracle") or {}).get("passed"):
                    direct["passed"] += 1
                else:
                    direct["failed"] += 1
    direct["success_rate"] = ratio(direct["passed"], direct["attempted"])
    direct["avg_duration_ms"] = average(direct["duration_ms"], direct["attempted"])
    return {BASELINE_DIRECT_CLI: direct}


def score(summary: dict[str, Any], mode: str) -> dict[str, Any]:
    total = summary.get("total") or {}
    direct_reuse_rate = ratio(total.get("oracle_pass_count", 0), total.get("held_out_reuses", 0))
    use_rate = ratio(total.get("use_oracle_pass_count", 0), total.get("held_out_uses", 0))
    closed_loop_rate = min(direct_reuse_rate, use_rate) if mode == MODE_REPLAY else use_rate
    return {
        "profile": mode,
        "closed_loop_success_rate": closed_loop_rate,
        "intent_use_success_rate": use_rate,
        "direct_reuse_success_rate": direct_reuse_rate if mode == MODE_REPLAY else None,
        "provider_yield_rate": ratio(total.get("providers_with_promoted_bindings", 0), total.get("provider_available", 0)),
        "probe_pass_rate": ratio(total.get("probe_pass_count", 0), total.get("candidate_count", 0)),
        "promotion_yield": ratio(total.get("promoted_bindings", 0), total.get("probe_pass_count", 0)),
        "provider_failure_rate": ratio(total.get("provider_failures", 0), total.get("provider_available", 0)),
        "negative_evidence_count": total.get("candidate_negative_evidence", 0),
        "negative_evidence_rate": ratio(total.get("candidate_negative_evidence", 0), total.get("candidate_count", 0)),
        "failed_count": total.get("failed", 0),
        "provider_acquisition_total_ms": total.get("acquisition_duration_ms", 0),
        "provider_acquisition_avg_ms": total.get("avg_acquisition_ms", 0),
        "proposal_llm_total_ms": total.get("llm_duration_ms", 0),
        "proposal_llm_avg_ms": total.get("avg_llm_ms", 0),
        "acquisition_local_overhead_total_ms": max(total.get("acquisition_duration_ms", 0) - total.get("llm_duration_ms", 0), 0),
        "intent_use_total_ms": total.get("use_duration_ms", 0),
        "intent_use_avg_ms": total.get("avg_use_ms", 0),
        "replay_direct_reuse_total_ms": total.get("reuse_duration_ms", 0) if mode == MODE_REPLAY else None,
        "replay_direct_reuse_avg_ms": total.get("avg_reuse_ms", 0) if mode == MODE_REPLAY else None,
    }


def capability_model(suites: dict[str, Any]) -> dict[str, Any]:
    providers: dict[str, set[str]] = {}
    capabilities: dict[str, set[str]] = {}
    for suite_data in suites.values():
        for case in suite_data.get("cases") or []:
            for provider in case.get("providers") or []:
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
        "multi_capability_providers": sum(1 for values in providers.values() if len(values) > 1),
        "multi_binding_capabilities": sum(1 for values in capabilities.values() if len(values) > 1),
    }


def failure_taxonomy(suites: dict[str, Any]) -> list[dict[str, Any]]:
    counts: dict[tuple[str, str], int] = {}
    for suite_data in suites.values():
        for case in suite_data.get("cases") or []:
            for provider in case.get("providers") or []:
                record_failure(counts, provider.get("failure"))
                for candidate in provider.get("candidates") or []:
                    record_failure(counts, (candidate.get("probe") or {}).get("error"))
                    for reuse in candidate.get("reuse") or []:
                        record_failure(counts, reuse.get("failure"))
                        record_failure(counts, reuse.get("skip"))
            for use in case.get("use") or []:
                record_failure(counts, use.get("failure"))
            for results in (case.get("baselines") or {}).values():
                for result in results:
                    record_failure(counts, result.get("failure"))
    return [{"stage": stage, "code": code, "count": count} for (stage, code), count in sorted(counts.items())]


def record_failure(counts: dict[tuple[str, str], int], value: Any) -> None:
    if not isinstance(value, dict):
        return
    stage = value.get("stage") or "unknown"
    code = value.get("code") or "unknown"
    counts[(stage, code)] = counts.get((stage, code), 0) + 1
