#!/usr/bin/env python3
from __future__ import annotations

import json
import sys
import tarfile


def main() -> int:
    request = json.load(sys.stdin)
    inputs = request.get("inputs") or {}
    target = inputs.get("target")
    if not target:
        return fail("missing_input", "target is required")
    if not tarfile.is_tarfile(target):
        return fail("tar_parse_failed", "target is not a readable TAR archive")
    with tarfile.open(target) as archive:
        names = [member.name for member in archive.getmembers() if member.isfile()]
    if not names:
        return fail("tar_empty", "target TAR archive has no file entries")
    return passed({"target": target, "entries": len(names)})


def passed(evidence: dict) -> int:
    print(json.dumps({"passed": True, "evidence": evidence}, separators=(",", ":")))
    return 0


def fail(code: str, message: str) -> int:
    print(json.dumps({"passed": False, "error": {"code": code, "message": message}}, separators=(",", ":")))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
