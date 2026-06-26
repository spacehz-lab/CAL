from __future__ import annotations

import hashlib
import json
import re
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


def new_run_id(now: datetime, mode: str, model: str) -> str:
    label = _clean_run_part(model or mode)
    digest = hashlib.sha1(f"{now.isoformat()}|{mode}|{model}".encode("utf-8")).hexdigest()[:6]
    return f"{now.strftime('%Y%m%d-%H%M%S')}-{label}-{digest}"


def default_out_dir(repo: Path) -> Path:
    return repo / "evals" / "out" / "experiments" / "cli-matrix"


def write_json(path: Path, value: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2, sort_keys=False) + "\n", encoding="utf-8")


def utc_now() -> datetime:
    return datetime.now(timezone.utc)


def _clean_run_part(value: str) -> str:
    value = value.strip().lower()
    value = re.sub(r"[^a-z0-9]+", "-", value)
    value = value.strip("-")
    return value or "run"
