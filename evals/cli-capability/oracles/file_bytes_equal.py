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

    with open(expected, "rb") as handle:
        expected_bytes = handle.read()
    with open(actual, "rb") as handle:
        actual_bytes = handle.read()
    if actual_bytes != expected_bytes:
        return fail("bytes_mismatch", "actual file bytes do not match expected file bytes")
    return passed({"expected": expected, "actual": actual, "bytes": len(actual_bytes)})


def passed(evidence: dict) -> int:
    print(json.dumps({"passed": True, "evidence": evidence}, separators=(",", ":")))
    return 0


def fail(code: str, message: str) -> int:
    print(json.dumps({"passed": False, "error": {"code": code, "message": message}}, separators=(",", ":")))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
