from __future__ import annotations

from pathlib import Path
from typing import Any

from constants import EXPERIMENTS, SCENARIO_GROUPS
from util import read_json, read_jsonl


class ScenarioCatalog:
    def __init__(self, bench: Path) -> None:
        self.bench = bench
        self.providers = {provider["id"]: provider for provider in read_json(bench / "providers.json")["providers"]}
        self.cases = self.load_cases()

    def select(
        self,
        experiments: str,
        level: str,
        case_names: str = "",
        provider_classes: str = "",
        scenario_tags: str = "",
        failure_types: str = "",
    ) -> list[dict[str, Any]]:
        wanted_experiments = parse_csv(experiments) or list(EXPERIMENTS)
        unknown_experiments = [experiment for experiment in wanted_experiments if experiment not in EXPERIMENTS]
        if unknown_experiments:
            raise SystemExit(f"unknown paper experiments: {', '.join(unknown_experiments)}")

        wanted_cases = set(parse_csv(case_names))
        wanted_provider_classes = set(parse_csv(provider_classes))
        wanted_tags = set(parse_csv(scenario_tags))
        wanted_failure_types = set(parse_csv(failure_types))
        selected: list[dict[str, Any]] = []
        for case in self.cases:
            case_experiments = set(case.get("paper_experiments") or [])
            if not case_experiments.intersection(wanted_experiments):
                continue
            if wanted_cases and case["id"] not in wanted_cases and case_key(case) not in wanted_cases:
                continue
            if level == "focus" and case.get("level") != "focus":
                continue
            if wanted_provider_classes and case.get("provider_class", "") not in wanted_provider_classes:
                continue
            if wanted_tags and not wanted_tags.intersection(set(case.get("scenario_tags") or [])):
                continue
            if wanted_failure_types and case.get("failure_type", "") not in wanted_failure_types:
                continue
            selected.append(self.with_providers(case))

        if not selected:
            raise SystemExit("no scenario cases selected")
        return selected

    def load_cases(self) -> list[dict[str, Any]]:
        rows: list[dict[str, Any]] = []
        seen: set[str] = set()
        for group in SCENARIO_GROUPS:
            path = self.bench / "scenarios" / f"{group}.jsonl"
            if not path.exists():
                raise SystemExit(f"missing scenario file {path}")
            for case in read_jsonl(path):
                case["scenario_group"] = group
                self.validate_case(path, case)
                key = case_key(case)
                if key in seen:
                    raise SystemExit(f"{path}: duplicate scenario case {key}")
                seen.add(key)
                rows.append(case)
        return rows

    def validate_case(self, path: Path, case: dict[str, Any]) -> None:
        for key in ("id", "intent", "provider_candidates", "oracle", "paper_experiments", "acquisition_mode", "scenario_tags"):
            if key not in case:
                raise SystemExit(f"{path}: missing required case key {key}")
        unknown_experiments = [experiment for experiment in case.get("paper_experiments") or [] if experiment not in EXPERIMENTS]
        if unknown_experiments:
            raise SystemExit(f"{path}: {case['id']}: unknown paper experiments: {', '.join(unknown_experiments)}")
        missing = [provider_id for provider_id in case["provider_candidates"] if provider_id not in self.providers]
        if missing:
            raise SystemExit(f"{path}: {case['id']}: unknown provider candidates: {', '.join(missing)}")

    def with_providers(self, case: dict[str, Any]) -> dict[str, Any]:
        copied = dict(case)
        copied["case_key"] = case_key(case)
        copied["providers"] = [self.providers[provider_id] for provider_id in case["provider_candidates"]]
        return copied


def case_key(case: dict[str, Any]) -> str:
    group = case.get("scenario_group", "")
    return f"{group}:{case.get('id', '')}" if group else case.get("id", "")


def parse_csv(value: str) -> list[str]:
    return [item.strip() for item in value.split(",") if item.strip()]
