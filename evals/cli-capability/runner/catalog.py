from __future__ import annotations

from pathlib import Path
from typing import Any

from constants import SUITES
from util import read_json, read_jsonl


class SuiteCatalog:
    def __init__(self, bench: Path) -> None:
        self.bench = bench
        self.providers = {provider["id"]: provider for provider in read_json(bench / "providers.json")["providers"]}
        self.cases = self.load_cases()

    def select(self, suite_names: str, level: str, case_names: str = "") -> list[dict[str, Any]]:
        suites = parse_csv(suite_names) or list(SUITES)
        unknown_suites = [suite for suite in suites if suite not in SUITES]
        if unknown_suites:
            raise SystemExit(f"unknown benchmark suites: {', '.join(unknown_suites)}")

        wanted_cases = set(parse_csv(case_names))
        selected: list[dict[str, Any]] = []
        for case in self.cases:
            if case["suite"] not in suites:
                continue
            if wanted_cases and case["id"] not in wanted_cases:
                continue
            if level == "focus" and case.get("level") != "focus":
                continue
            selected.append(self.with_providers(case))

        if not selected:
            raise SystemExit("no suite cases selected")
        return selected

    def load_cases(self) -> list[dict[str, Any]]:
        rows: list[dict[str, Any]] = []
        for suite in SUITES:
            path = self.bench / "suites" / f"{suite}.jsonl"
            if not path.exists():
                raise SystemExit(f"missing suite file {path}")
            for case in read_jsonl(path):
                actual = case.get("suite")
                if actual != suite:
                    raise SystemExit(f"{path}: case {case.get('id', '')} has suite {actual!r}, want {suite!r}")
                self.validate_case(path, case)
                rows.append(case)
        return rows

    def validate_case(self, path: Path, case: dict[str, Any]) -> None:
        for key in ("id", "suite", "intent", "provider_candidates", "oracle"):
            if key not in case:
                raise SystemExit(f"{path}: missing required case key {key}")
        missing = [provider_id for provider_id in case["provider_candidates"] if provider_id not in self.providers]
        if missing:
            raise SystemExit(f"{path}: {case['id']}: unknown provider candidates: {', '.join(missing)}")

    def with_providers(self, case: dict[str, Any]) -> dict[str, Any]:
        copied = dict(case)
        copied["providers"] = [self.providers[provider_id] for provider_id in case["provider_candidates"]]
        return copied


def parse_csv(value: str) -> list[str]:
    return [item.strip() for item in value.split(",") if item.strip()]
