#!/usr/bin/env python3
from __future__ import annotations

import json
import py_compile
from pathlib import Path

from constants import (
    ACQUISITION_FULL,
    BASELINE_LLM_ONESHOT,
    EXPERIMENTS,
    EXPERIMENT_ACQUISITION,
    EXPERIMENT_REPEATED_REUSE,
    SCENARIO_GROUPS,
    TAG_REUSE_COMPARISON,
)


ROOT = Path(__file__).resolve().parents[1]
IMPLEMENTED_BASELINES = {BASELINE_LLM_ONESHOT}


def main() -> int:
    cases = []
    for group in SCENARIO_GROUPS:
        path = ROOT / "scenarios" / f"{group}.jsonl"
        if not path.exists():
            raise SystemExit(f"missing scenario file {path}")
        for case in read_jsonl(path):
            case["scenario_group"] = group
            cases.append(case)
    providers = read_json(ROOT / "providers.json")
    provider_ids = {provider["id"] for provider in providers["providers"]}

    seen_cases: set[tuple[str, str]] = set()
    for case in cases:
        case_id = require(case, "id")
        group = require(case, "scenario_group")
        key = (group, case_id)
        if key in seen_cases:
            raise SystemExit(f"duplicate scenario case id: {group}:{case_id}")
        seen_cases.add(key)
        require(case, "intent")
        experiments = require(case, "paper_experiments")
        for experiment in experiments:
            if experiment not in EXPERIMENTS:
                raise SystemExit(f"{group}:{case_id}: unknown experiment {experiment}")
        check_baselines(case_id, experiments, case.get("baselines") or [])
        for provider in require(case, "provider_candidates"):
            if provider not in provider_ids:
                raise SystemExit(f"{group}:{case_id}: unknown provider candidate {provider}")
        oracle = require(case, "oracle")
        oracle_path = ROOT / require(oracle, "path")
        if not oracle_path.exists():
            raise SystemExit(f"{group}:{case_id}: missing oracle {oracle_path}")
        py_compile.compile(str(oracle_path), doraise=True)
        check_fixture_group(group, case_id, case.get("acquisition") or {}, "fixtures")
        check_expected_capabilities(group, case_id, case)
        if EXPERIMENT_REPEATED_REUSE in experiments:
            check_fixture_group(group, case_id, case.get("reuse") or {}, "rounds")
            check_reuse_comparison_case(group, case_id, case)
        if case.get("level") == "focus":
            for provider in case["provider_candidates"]:
                proposal_path = ROOT / "proposals" / "replay" / case_id / f"{provider}.json"
                if not proposal_path.exists():
                    raise SystemExit(f"{group}:{case_id}: missing focus replay proposal {proposal_path}")
                check_replay_proposal(proposal_path)

    for path in (ROOT / "oracles").glob("*.py"):
        py_compile.compile(str(path), doraise=True)
    for path in (ROOT / "proposals" / "replay").glob("*/*.json"):
        check_replay_proposal(path)
    check_full_acquisition_design(cases)
    check_reuse_comparison_design(cases)
    print(f"validated {len(cases)} scenario cases and {len(provider_ids)} providers")
    return 0


def check_baselines(case_id: str, experiments: list[str], baselines: list[str]) -> None:
    if baselines and EXPERIMENT_REPEATED_REUSE not in experiments:
        raise SystemExit(f"{case_id}: baselines are only allowed for repeated_reuse experiment")
    for baseline in baselines:
        if baseline not in IMPLEMENTED_BASELINES:
            raise SystemExit(f"{case_id}: baseline {baseline!r} is not implemented")


def check_fixture_group(group: str, case_id: str, fixture_group: dict, key: str) -> None:
    fixtures = fixture_group.get(key) or []
    if not fixtures:
        raise SystemExit(f"{group}:{case_id}: {key} group is empty")
    for fixture in fixtures:
        inputs = fixture.get("inputs") or {}
        for input_key, value in inputs.items():
            if not isinstance(value, str):
                continue
            if value.startswith("{work}/"):
                continue
            if value.startswith("fixtures/") and not (ROOT / value).exists():
                raise SystemExit(f"{group}:{case_id}: missing fixture input {input_key}={value}")


def check_expected_capabilities(group: str, case_id: str, case: dict) -> None:
    expected = case.get("expected_capabilities") or []
    for index, item in enumerate(expected):
        if not isinstance(item, dict):
            raise SystemExit(f"{group}:{case_id}: expected_capabilities[{index}] must be an object")
        for key in ("key", "surface", "min_promoted_bindings"):
            if key not in item:
                raise SystemExit(f"{group}:{case_id}: expected_capabilities[{index}] missing {key}")
        if not isinstance(item["min_promoted_bindings"], int) or item["min_promoted_bindings"] < 1:
            raise SystemExit(f"{group}:{case_id}: expected_capabilities[{index}].min_promoted_bindings must be a positive integer")


def check_full_acquisition_design(cases: list[dict]) -> None:
    full_cases = [case for case in cases if case.get("acquisition_mode") == ACQUISITION_FULL and EXPERIMENT_ACQUISITION in (case.get("paper_experiments") or [])]
    if not full_cases:
        return
    multi_cap_cases = [case for case in full_cases if len(case.get("expected_capabilities") or []) >= 2]
    if len(multi_cap_cases) / len(full_cases) < 0.8:
        raise SystemExit(
            "full_acquisition design requires at least 80% provider-suite cases "
            f"with multiple expected capabilities; got {len(multi_cap_cases)}/{len(full_cases)}"
        )


def check_reuse_comparison_case(group: str, case_id: str, case: dict) -> None:
    if TAG_REUSE_COMPARISON not in (case.get("scenario_tags") or []):
        return
    if BASELINE_LLM_ONESHOT not in (case.get("baselines") or []):
        raise SystemExit(f"{group}:{case_id}: reuse comparison cases must include llm_oneshot baseline")
    provider = case.get("baseline_provider", "")
    if not provider:
        raise SystemExit(f"{group}:{case_id}: reuse comparison cases require baseline_provider")
    if provider not in (case.get("provider_candidates") or []):
        raise SystemExit(f"{group}:{case_id}: baseline_provider {provider!r} is not a provider candidate")


def check_reuse_comparison_design(cases: list[dict]) -> None:
    comparison = [
        case
        for case in cases
        if EXPERIMENT_REPEATED_REUSE in (case.get("paper_experiments") or [])
        and TAG_REUSE_COMPARISON in (case.get("scenario_tags") or [])
    ]
    if not comparison:
        return
    rounds = sum(len((case.get("reuse") or {}).get("rounds") or []) for case in comparison)
    if len(comparison) != 8 or rounds != 10:
        raise SystemExit(f"reuse comparison design requires 8 cases and 10 rounds; got {len(comparison)} cases and {rounds} rounds")


def check_replay_proposal(path: Path) -> None:
    proposal = read_json(path)
    candidates = proposal.get("candidates") or []
    if not candidates:
        raise SystemExit(f"{path}: replay proposal must include candidates")
    plans = proposal.get("probe_plans") or []
    if not plans:
        raise SystemExit(f"{path}: replay proposal must include probe_plans")
    for index, plan in enumerate(plans):
        candidate_index = plan.get("candidate_index")
        if not isinstance(candidate_index, int) or candidate_index < 0 or candidate_index >= len(candidates):
            raise SystemExit(f"{path}: probe_plan {index} candidate_index is out of range")
        verify = plan.get("verify")
        if not isinstance(verify, dict):
            raise SystemExit(f"{path}: probe_plan {index} verify is required")
        check_verify_spec(path, index, verify)


def check_verify_spec(path: Path, index: int, verify: dict) -> None:
    level = verify.get("level")
    method = verify.get("method")
    if level not in {"L0", "L1", "L2", "L3"}:
        raise SystemExit(f"{path}: probe_plan {index} verify level {level!r} is invalid")
    if method not in {"execute", "contract"}:
        raise SystemExit(f"{path}: probe_plan {index} verify method {method!r} is invalid")
    checks = verify.get("checks") or []
    if method == "contract" and checks:
        raise SystemExit(f"{path}: probe_plan {index} contract verify cannot include checks")
    if method == "execute" and level != "L0" and not checks:
        raise SystemExit(f"{path}: probe_plan {index} execute verify requires checks")
    for check_index, check in enumerate(checks):
        subject = check.get("subject") or {}
        subject_type = subject.get("type")
        if subject_type not in {"file", "stdout", "stderr", "exit_code"}:
            raise SystemExit(f"{path}: probe_plan {index} check {check_index} subject type is invalid")
        if subject_type == "file" and not subject.get("input"):
            raise SystemExit(f"{path}: probe_plan {index} check {check_index} file subject requires input")
        predicate = check.get("predicate")
        if predicate not in verify_predicates():
            raise SystemExit(f"{path}: probe_plan {index} check {check_index} predicate {predicate!r} is invalid")
        params = check.get("params") or {}
        for param in required_params(predicate):
            if param not in params:
                raise SystemExit(f"{path}: probe_plan {index} check {check_index} predicate {predicate} requires params.{param}")


def required_params(predicate: str) -> list[str]:
    return {
        "equals": ["value"],
        "not_equals": ["value"],
        "contains": ["value"],
        "contains_any": ["values"],
        "regex": ["pattern"],
        "format": ["format"],
        "bytes_equal_transform": ["source", "transform"],
        "hash_line_matches": ["source", "algorithm"],
        "archive_contains_input": ["source", "format"],
        "json_query_matches": ["source", "query"],
        "json_equivalent": ["source"],
        "json_field_equals": ["query", "value"],
        "json_field_matches_source": ["query", "source", "property"],
        "text_transform_matches": ["source", "transform"],
        "line_count_matches": ["source"],
        "text_filter_matches": ["source", "pattern"],
        "delimited_column_matches": ["source", "delimiter", "column"],
    }.get(predicate, [])


def verify_predicates() -> set[str]:
    return {
        "equals",
        "not_equals",
        "exists",
        "non_empty",
        "format",
        "contains",
        "contains_any",
        "regex",
        "bytes_equal_transform",
        "hash_line_matches",
        "archive_contains_input",
        "json_query_matches",
        "json_equivalent",
        "json_field_equals",
        "json_field_matches_source",
        "text_transform_matches",
        "line_count_matches",
        "text_filter_matches",
        "delimited_column_matches",
    }


def require(doc: dict, key: str):
    if key not in doc:
        raise SystemExit(f"missing required key {key}")
    return doc[key]


def read_json(path: Path):
    with path.open("r", encoding="utf-8") as handle:
        return json.load(handle)


def read_jsonl(path: Path) -> list[dict]:
    rows = []
    with path.open("r", encoding="utf-8") as handle:
        for line_no, line in enumerate(handle, 1):
            if not line.strip():
                continue
            try:
                rows.append(json.loads(line))
            except json.JSONDecodeError as exc:
                raise SystemExit(f"{path}:{line_no}: {exc}") from exc
    return rows


if __name__ == "__main__":
    raise SystemExit(main())
