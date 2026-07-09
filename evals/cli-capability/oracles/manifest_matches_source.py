#!/usr/bin/env python3
from __future__ import annotations

import hashlib
import json
import sys
from pathlib import Path


def main() -> int:
    request = json.load(sys.stdin)
    inputs = request.get("inputs") or {}
    source = inputs.get("source")
    target = inputs.get("target")
    label = inputs.get("label")
    if not source or not target or not label:
        return fail("missing_input", "source, target, and label are required")
    source_path = Path(source)
    content = source_path.read_bytes()
    with open(target, "r", encoding="utf-8") as handle:
        actual = json.load(handle)
    expected = {
        "basename": source_path.name,
        "bytes": len(content),
        "label": label,
        "sha256": hashlib.sha256(content).hexdigest(),
    }
    if actual != expected:
        return fail("manifest_mismatch", "manifest does not match source file")
    return passed({"source": source, "target": target, "label": label})


def passed(evidence: dict) -> int:
    print(json.dumps({"passed": True, "evidence": evidence}, separators=(",", ":")))
    return 0


def fail(code: str, message: str) -> int:
    print(json.dumps({"passed": False, "error": {"code": code, "message": message}}, separators=(",", ":")))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
