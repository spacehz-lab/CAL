#!/usr/bin/env python3
from __future__ import annotations

import json
import sys
import zipfile


def main() -> int:
    request = json.load(sys.stdin)
    inputs = request.get("inputs") or {}
    target = inputs.get("target")
    if not target:
        return fail("missing_input", "target is required")
    if not zipfile.is_zipfile(target):
        return fail("zip_parse_failed", "target is not a readable ZIP archive")
    with zipfile.ZipFile(target) as archive:
        names = [name for name in archive.namelist() if name and not name.endswith("/")]
    if not names:
        return fail("zip_empty", "target ZIP archive has no file entries")
    return passed({"target": target, "entries": len(names)})


def passed(evidence: dict) -> int:
    print(json.dumps({"passed": True, "evidence": evidence}, separators=(",", ":")))
    return 0


def fail(code: str, message: str) -> int:
    print(json.dumps({"passed": False, "error": {"code": code, "message": message}}, separators=(",", ":")))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
