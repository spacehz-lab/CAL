#!/usr/bin/env python3
from __future__ import annotations

import hashlib
import json
import re
import sys


def main() -> int:
    request = json.load(sys.stdin)
    inputs = request.get("inputs") or {}
    source = inputs.get("source")
    target = inputs.get("target")
    if not source or not target:
        return fail("missing_input", "source and target are required")
    digest = hashlib.sha256()
    with open(source, "rb") as handle:
        digest.update(handle.read())
    expected = digest.hexdigest()
    with open(target, "r", encoding="utf-8", errors="ignore") as handle:
        content = handle.read()
    match = re.search(r"\b[0-9a-fA-F]{64}\b", content)
    if not match:
        return fail("missing_sha256", "target does not contain a SHA-256 digest")
    if match.group(0).lower() != expected:
        return fail("sha256_mismatch", "target digest does not match source")
    return passed({"algorithm": "sha256", "source": source, "target": target})


def passed(evidence: dict) -> int:
    print(json.dumps({"passed": True, "evidence": evidence}, separators=(",", ":")))
    return 0


def fail(code: str, message: str) -> int:
    print(json.dumps({"passed": False, "error": {"code": code, "message": message}}, separators=(",", ":")))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
