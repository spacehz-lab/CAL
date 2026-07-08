#!/usr/bin/env python3
from __future__ import annotations

import json
import py_compile
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
SUITES = {"acquisition", "capability_model", "reuse"}
IMPLEMENTED_BASELINES = {"direct_cli"}


def main() -> int:
    cases = []
    for suite in sorted(SUITES):
        path = ROOT / "suites" / f"{suite}.jsonl"
        if not path.exists():
            raise SystemExit(f"missing suite file {path}")
        for case in read_jsonl(path):
            if case.get("suite") != suite:
                raise SystemExit(f"{path}: case {case.get('id', '')} has suite {case.get('suite')!r}, want {suite!r}")
            cases.append(case)
    providers = read_json(ROOT / "providers.json")
    provider_ids = {provider["id"] for provider in providers["providers"]}

    seen_cases: set[tuple[str, str]] = set()
    for case in cases:
        case_id = require(case, "id")
        suite = require(case, "suite")
        key = (suite, case_id)
        if key in seen_cases:
            raise SystemExit(f"duplicate suite case id: {suite}:{case_id}")
        seen_cases.add(key)
        require(case, "intent")
        check_baselines(case_id, suite, case.get("baselines") or [])
        for provider in require(case, "provider_candidates"):
            if provider not in provider_ids:
                raise SystemExit(f"{case_id}: unknown provider candidate {provider}")
        oracle = require(case, "oracle")
        oracle_path = ROOT / require(oracle, "path")
        if not oracle_path.exists():
            raise SystemExit(f"{case_id}: missing oracle {oracle_path}")
        py_compile.compile(str(oracle_path), doraise=True)
        check_fixture_group(case_id, case.get("acquisition") or {})
        check_fixture_group(case_id, case.get("reuse") or {})
        if case.get("level") == "focus":
            for provider in case["provider_candidates"]:
                proposal_path = ROOT / "proposals" / "replay" / case_id / f"{provider}.json"
                if not proposal_path.exists():
                    raise SystemExit(f"{case_id}: missing focus replay proposal {proposal_path}")
                check_replay_proposal(proposal_path)

    for path in (ROOT / "oracles").glob("*.py"):
        py_compile.compile(str(path), doraise=True)
    for path in (ROOT / "proposals" / "replay").glob("*/*.json"):
        check_replay_proposal(path)
    read_jsonl(ROOT / "baselines" / "oracle" / "commands.jsonl")
    print(f"validated {len(cases)} cases and {len(provider_ids)} providers")
    return 0


def check_baselines(case_id: str, suite: str, baselines: list[str]) -> None:
    if baselines and suite != "reuse":
        raise SystemExit(f"{case_id}: baselines are only allowed in reuse suite")
    for baseline in baselines:
        if baseline not in IMPLEMENTED_BASELINES:
            raise SystemExit(f"{case_id}: baseline {baseline!r} is not implemented")


def check_fixture_group(case_id: str, group: dict) -> None:
    fixtures = group.get("fixtures") or []
    if not fixtures:
        raise SystemExit(f"{case_id}: fixture group is empty")
    for fixture in fixtures:
        inputs = fixture.get("inputs") or {}
        for key, value in inputs.items():
            if not isinstance(value, str):
                continue
            if value.startswith("{work}/"):
                continue
            if value.startswith("fixtures/") and not (ROOT / value).exists():
                raise SystemExit(f"{case_id}: missing fixture input {key}={value}")


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
        required = required_params(predicate)
        for key in required:
            if key not in params:
                raise SystemExit(f"{path}: probe_plan {index} check {check_index} predicate {predicate} requires params.{key}")


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
