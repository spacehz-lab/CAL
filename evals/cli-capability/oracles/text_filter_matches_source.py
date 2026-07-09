#!/usr/bin/env python3
from __future__ import annotations

import json
import sys


def main() -> int:
    request = json.load(sys.stdin)
    inputs = request.get("inputs") or {}
    source = inputs.get("source")
    target = inputs.get("target")
    pattern = inputs.get("pattern")
    if not source or not target or not pattern:
        return fail("missing_input", "source, target, and pattern are required")
    with open(source, "r", encoding="utf-8", errors="ignore") as handle:
        expected = [line.rstrip("\r\n") for line in handle if pattern in line]
    with open(target, "r", encoding="utf-8", errors="ignore") as handle:
        actual = [line.rstrip("\r\n") for line in handle if line.rstrip("\r\n")]
    if actual != expected:
        return fail("filter_mismatch", "target filtered lines do not match source")
    return passed({"source": source, "target": target, "pattern": pattern, "lines": len(expected)})


def passed(evidence: dict) -> int:
    print(json.dumps({"passed": True, "evidence": evidence}, separators=(",", ":")))
    return 0


def fail(code: str, message: str) -> int:
    print(json.dumps({"passed": False, "error": {"code": code, "message": message}}, separators=(",", ":")))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
