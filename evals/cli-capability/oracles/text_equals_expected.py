#!/usr/bin/env python3
from __future__ import annotations

import json
import sys


def main() -> int:
    request = json.load(sys.stdin)
    inputs = request.get("inputs") or {}
    oracle = request.get("oracle") or {}
    expected_key = oracle.get("expected_input") or "expected"
    actual_key = oracle.get("actual_input") or "target"
    expected = inputs.get(expected_key)
    actual = inputs.get(actual_key)
    if not expected or not actual:
        return fail("missing_input", f"{expected_key} and {actual_key} are required")
    with open(expected, "r", encoding="utf-8", errors="ignore") as handle:
        expected_text = handle.read().strip()
    with open(actual, "r", encoding="utf-8", errors="ignore") as handle:
        actual_text = handle.read().strip()
    if actual_text != expected_text:
        return fail("text_mismatch", "actual text does not match expected text")
    return passed({"expected": expected, "actual": actual})


def passed(evidence: dict) -> int:
    print(json.dumps({"passed": True, "evidence": evidence}, separators=(",", ":")))
    return 0


def fail(code: str, message: str) -> int:
    print(json.dumps({"passed": False, "error": {"code": code, "message": message}}, separators=(",", ":")))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
