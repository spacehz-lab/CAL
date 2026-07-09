#!/usr/bin/env python3
from __future__ import annotations

import json
import sys


def main() -> int:
    request = json.load(sys.stdin)
    inputs = request.get("inputs") or {}
    source = inputs.get("source")
    target = inputs.get("target")
    delimiter = inputs.get("delimiter")
    column = inputs.get("column")
    if not source or not target or delimiter is None or column is None:
        return fail("missing_input", "source, target, delimiter, and column are required")
    column_index = int(column) - 1
    expected = []
    with open(source, "r", encoding="utf-8", errors="ignore") as handle:
        for line in handle:
            parts = line.rstrip("\r\n").split(delimiter)
            if 0 <= column_index < len(parts):
                expected.append(parts[column_index])
    with open(target, "r", encoding="utf-8", errors="ignore") as handle:
        actual = [line.rstrip("\r\n") for line in handle if line.rstrip("\r\n")]
    if actual != expected:
        return fail("column_mismatch", "target column values do not match source")
    return passed({"source": source, "target": target, "column": column})


def passed(evidence: dict) -> int:
    print(json.dumps({"passed": True, "evidence": evidence}, separators=(",", ":")))
    return 0


def fail(code: str, message: str) -> int:
    print(json.dumps({"passed": False, "error": {"code": code, "message": message}}, separators=(",", ":")))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
