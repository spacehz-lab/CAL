#!/usr/bin/env python3
from __future__ import annotations

import json
import sys


def main() -> int:
    request = json.load(sys.stdin)
    inputs = request.get("inputs") or {}
    source = inputs.get("source")
    target = inputs.get("target")
    if not source or not target:
        return fail("missing_input", "source and target are required")
    with open(source, "r", encoding="utf-8") as handle:
        expected = json.load(handle)
    with open(target, "r", encoding="utf-8") as handle:
        actual = json.load(handle)
    if actual != expected:
        return fail("json_mismatch", "target JSON is not semantically equivalent to source")
    return passed({"source": source, "target": target})


def passed(evidence: dict) -> int:
    print(json.dumps({"passed": True, "evidence": evidence}, separators=(",", ":")))
    return 0


def fail(code: str, message: str) -> int:
    print(json.dumps({"passed": False, "error": {"code": code, "message": message}}, separators=(",", ":")))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
