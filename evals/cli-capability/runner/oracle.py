from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path
from typing import Any

from util import parse_json


class OracleRunner:
    def __init__(self, repo: Path, bench: Path) -> None:
        self.repo = repo
        self.bench = bench

    def run(self, case: dict[str, Any], inputs: dict[str, Any]) -> dict[str, Any]:
        oracle = case["oracle"]
        oracle_path = self.bench / oracle["path"]
        request = {"inputs": inputs, "oracle": oracle}
        completed = subprocess.run(
            [sys.executable, str(oracle_path)],
            cwd=self.repo,
            input=json.dumps(request),
            text=True,
            capture_output=True,
        )
        parsed = parse_json(completed.stdout)
        if completed.returncode != 0 or not parsed:
            return {
                "passed": False,
                "error": {
                    "code": "oracle_failed",
                    "message": completed.stderr.strip() or completed.stdout.strip() or f"exit status {completed.returncode}",
                },
            }
        parsed["id"] = oracle.get("id", "")
        return parsed
