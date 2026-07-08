from __future__ import annotations

import hashlib
import html
import json
import re
import time
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


def read_json(path: Path) -> dict[str, Any]:
    return json.loads(path.read_text(encoding="utf-8"))


def read_jsonl(path: Path) -> list[dict[str, Any]]:
    rows: list[dict[str, Any]] = []
    with path.open("r", encoding="utf-8") as handle:
        for line_no, line in enumerate(handle, 1):
            if not line.strip():
                continue
            try:
                value = json.loads(line)
            except json.JSONDecodeError as exc:
                raise SystemExit(f"{path}:{line_no}: {exc}") from exc
            if not isinstance(value, dict):
                raise SystemExit(f"{path}:{line_no}: expected JSON object")
            rows.append(value)
    return rows


def parse_json(text: str) -> dict[str, Any] | None:
    try:
        value = json.loads(text)
    except json.JSONDecodeError:
        return None
    return value if isinstance(value, dict) else None


def write_json(path: Path, value: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2) + "\n", encoding="utf-8")


def flow_step(name: str, status: str, **fields: Any) -> dict[str, Any]:
    item: dict[str, Any] = {"name": name, "status": status}
    for key, value in fields.items():
        if value in ("", None, [], {}):
            continue
        item[key] = value
    return item


def failure(stage: str, code: str, message: str) -> dict[str, str]:
    return {"stage": stage, "code": code, "message": message}


def strip_private_fields(value: Any) -> Any:
    if isinstance(value, dict):
        return {key: strip_private_fields(item) for key, item in value.items() if not key.startswith("_")}
    if isinstance(value, list):
        return [strip_private_fields(item) for item in value]
    return value


def elapsed_ms(started: float) -> int:
    return int((time.monotonic() - started) * 1000)


def average(total: int, count: int) -> int:
    return int(total / count) if count else 0


def ratio(numerator: int | float, denominator: int | float) -> float:
    return float(numerator) / float(denominator) if denominator else 0.0


def new_run_id(mode: str, model: str = "") -> str:
    now = datetime.now(timezone.utc)
    label = clean_run_part(model or mode)
    digest = hashlib.sha1(f"{now.isoformat()}|{mode}|{model}".encode("utf-8")).hexdigest()[:6]
    return f"{now.strftime('%Y%m%d-%H%M%S')}-{label}-{digest}"


def clean_run_part(value: str) -> str:
    cleaned = "".join(char.lower() if char.isalnum() else "-" for char in value.strip())
    cleaned = "-".join(part for part in cleaned.split("-") if part)
    return cleaned[:40] or "run"


def template_inputs(value: str) -> list[str]:
    return re.findall(r"{{\s*([A-Za-z_][A-Za-z0-9_]*)\s*}}", value)


def render_template(value: str, inputs: dict[str, Any]) -> str:
    rendered = value
    for name in template_inputs(value):
        rendered = re.sub(r"{{\s*" + re.escape(name) + r"\s*}}", str(inputs.get(name, "")), rendered)
    return rendered


def escape(value: Any) -> str:
    return html.escape(str(value if value is not None else ""))


def format_duration_ms(value: int) -> str:
    if value >= 60_000:
        return f"{value / 60_000:.1f} min"
    if value >= 1_000:
        return f"{value / 1_000:.1f} s"
    return f"{value} ms"


def format_duration_value(value: Any) -> str:
    if value is None:
        return "n/a"
    if isinstance(value, int):
        return format_duration_ms(value)
    return str(value)


def fraction(numerator: Any, denominator: Any) -> str:
    numerator_value = int(numerator or 0)
    denominator_value = int(denominator or 0)
    if denominator_value == 0:
        return "0/0"
    return f"{numerator_value}/{denominator_value} ({ratio(numerator_value, denominator_value) * 100:.1f}%)"
