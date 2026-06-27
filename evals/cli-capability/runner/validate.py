#!/usr/bin/env python3
from __future__ import annotations

import json
import py_compile
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


def main() -> int:
    tasks = read_jsonl(ROOT / "tasks.jsonl")
    providers = read_json(ROOT / "providers.json")
    provider_ids = {provider["id"] for provider in providers["providers"]}

    seen_tasks: set[str] = set()
    for task in tasks:
        task_id = require(task, "id")
        if task_id in seen_tasks:
            raise SystemExit(f"duplicate task id: {task_id}")
        seen_tasks.add(task_id)
        require(task, "intent")
        for provider in require(task, "provider_candidates"):
            if provider not in provider_ids:
                raise SystemExit(f"{task_id}: unknown provider candidate {provider}")
        oracle = require(task, "oracle")
        oracle_path = ROOT / require(oracle, "path")
        if not oracle_path.exists():
            raise SystemExit(f"{task_id}: missing oracle {oracle_path}")
        py_compile.compile(str(oracle_path), doraise=True)
        check_fixture_group(task_id, task.get("acquisition") or {})
        check_fixture_group(task_id, task.get("reuse") or {})

    for path in (ROOT / "oracles").glob("*.py"):
        py_compile.compile(str(path), doraise=True)
    for path in (ROOT / "proposals" / "replay").glob("*/*.json"):
        read_json(path)
    read_jsonl(ROOT / "baselines" / "oracle" / "commands.jsonl")
    print(f"validated {len(tasks)} tasks and {len(provider_ids)} providers")
    return 0


def check_fixture_group(task_id: str, group: dict) -> None:
    fixtures = group.get("fixtures") or []
    if not fixtures:
        raise SystemExit(f"{task_id}: fixture group is empty")
    for fixture in fixtures:
        inputs = fixture.get("inputs") or {}
        for key, value in inputs.items():
            if not isinstance(value, str):
                continue
            if value.startswith("{work}/"):
                continue
            if value.startswith("fixtures/") and not (ROOT / value).exists():
                raise SystemExit(f"{task_id}: missing fixture input {key}={value}")


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
