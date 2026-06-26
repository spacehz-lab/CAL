#!/usr/bin/env python3
from __future__ import annotations

import base64
import json
import sys


def main() -> int:
    request = json.load(sys.stdin)
    inputs = request.get("inputs") or {}
    source = inputs.get("source")
    target = inputs.get("target")
    if not source or not target:
        return fail("missing_input", "source and target are required")

    with open(source, "rb") as handle:
        expected = base64.b64encode(handle.read()).decode("ascii")
    with open(target, "r", encoding="utf-8", errors="ignore") as handle:
        actual = "".join(handle.read().split())
    if actual != expected:
        return fail("base64_mismatch", "target is not the Base64 encoding of source")
    return passed({"source": source, "target": target})


def passed(evidence: dict) -> int:
    print(json.dumps({"passed": True, "evidence": evidence}, separators=(",", ":")))
    return 0


def fail(code: str, message: str) -> int:
    print(json.dumps({"passed": False, "error": {"code": code, "message": message}}, separators=(",", ":")))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
