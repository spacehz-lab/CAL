from __future__ import annotations

import html
import json
from pathlib import Path
from typing import Any


def write_html(path: Path, artifact: dict[str, Any], template_path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    template = template_path.read_text(encoding="utf-8")
    path.write_text(
        template.replace("{{SUMMARY_CARDS}}", _summary_cards(artifact))
        .replace("{{CASE_ROWS}}", _case_rows(artifact))
        .replace("{{ARTIFACT_JSON}}", html.escape(json.dumps(_safe_header(artifact), indent=2))),
        encoding="utf-8",
    )


def duration(ms: int | float | None) -> str:
    if not ms:
        return "0 ms"
    ms = int(ms)
    if ms < 1000:
        return f"{ms} ms"
    if ms < 60000:
        return _duration_unit(ms / 1000, "s")
    return _duration_unit(ms / 60000, "min")


def _duration_unit(value: float, unit: str) -> str:
    text = f"{value:.1f}".rstrip("0").rstrip(".")
    return f"{text} {unit}"


def _summary_cards(artifact: dict[str, Any]) -> str:
    summary = artifact.get("summary") or {}
    cards = [
        ("CLI attempted", summary.get("cli_attempted", 0)),
        ("Candidates", summary.get("candidate_count", 0)),
        ("Promoted", summary.get("promoted_bindings", 0)),
        ("Verified reuse", summary.get("verified_reuses", 0)),
        ("Failed", summary.get("failed", 0)),
        ("LLM duration", duration(summary.get("llm_duration_ms", 0))),
    ]
    return "\n".join(
        f'<div class="card">{html.escape(label)}<br><strong>{html.escape(str(value))}</strong></div>'
        for label, value in cards
    )


def _case_rows(artifact: dict[str, Any]) -> str:
    rows: list[str] = []
    for case in artifact.get("cases") or []:
        candidates = case.get("candidates") or []
        if not candidates:
            rows.append(_empty_case_row(case))
            continue
        for candidate in candidates:
            rows.append(_candidate_row(case, candidate))
    return "\n".join(rows)


def _candidate_row(case: dict[str, Any], candidate: dict[str, Any]) -> str:
    probe = candidate.get("probe") or {}
    promotion = candidate.get("promotion") or {}
    reuse = candidate.get("reuse") or {}
    failure = candidate.get("failure") or case.get("failure")
    probe_status = '<span class="ok">passed</span>' if probe.get("passed") else f'<span class="fail">{_e(probe.get("status", "not_run"))}</span>'
    promotion_status = '<span class="muted">none</span>'
    if promotion:
        promotion_status = f'<span class="ok">{_e(promotion.get("capability_action", ""))}/{_e(promotion.get("binding_action", ""))}</span><br><code>{_e(promotion.get("binding_id", ""))}</code>'
    reuse_status = '<span class="muted">not run</span>'
    if reuse:
        if reuse.get("verified"):
            reuse_status = f'<span class="ok">verified</span><br><span class="muted">{duration(reuse.get("duration_ms"))}</span>'
        else:
            reuse_failure = reuse.get("failure") or {}
            reuse_status = f'<span class="fail">{_e(reuse_failure.get("stage", reuse.get("status", "failed")))}</span><br><span class="muted">{duration(reuse.get("duration_ms"))}</span>'
    failure_text = '<span class="ok">none</span>'
    if failure:
        failure_text = f'<span class="fail">{_e(failure.get("stage", ""))}</span><br><code>{_e(failure.get("code", ""))}</code><br>{_e(failure.get("message", ""))}'
    inputs = "".join(
        f'<code>{_e(item.get("key", ""))}={_e(item.get("value", ""))}</code><br>'
        for item in candidate.get("probe_inputs") or []
    )
    return f"""<tr>
<td><strong>{_e(case.get("cli", ""))}</strong><br><span class="muted">{_e(case.get("name", ""))}</span><br><code>{_e(case.get("provider_id", ""))}</code></td>
<td><code>{_e(candidate.get("capability_id", ""))}</code><br>{_e(candidate.get("description", ""))}<details><summary>execution</summary><code>{_e(json.dumps(candidate.get("execution", {}), sort_keys=True))}</code></details></td>
<td><code>{_e(candidate.get("verifier_id", ""))}</code><br><span class="muted">{_e(candidate.get("verifier_source", ""))}</span></td>
<td>{probe_status}<details><summary>inputs</summary>{inputs}</details></td>
<td>{promotion_status}</td>
<td>{reuse_status}</td>
<td>{failure_text}</td>
<td>scan {duration(case.get("scan_duration_ms"))}<br>llm {duration(case.get("llm_duration_ms"))}</td>
</tr>"""


def _empty_case_row(case: dict[str, Any]) -> str:
    failure = case.get("failure") or {}
    failure_text = '<span class="muted">none</span>'
    if failure:
        failure_text = f'<span class="fail">{_e(failure.get("stage", ""))}</span><br><code>{_e(failure.get("code", ""))}</code><br>{_e(failure.get("message", ""))}'
    return f"""<tr>
<td><strong>{_e(case.get("cli", ""))}</strong><br><span class="muted">{_e(case.get("name", ""))}</span></td>
<td colspan="5" class="muted">no candidates</td>
<td>{failure_text}</td>
<td>scan {duration(case.get("scan_duration_ms"))}<br>llm {duration(case.get("llm_duration_ms"))}</td>
</tr>"""


def _safe_header(artifact: dict[str, Any]) -> dict[str, Any]:
    return {
        "run_id": artifact.get("run_id"),
        "status": artifact.get("status"),
        "mode": artifact.get("mode"),
        "level": artifact.get("level"),
        "selected_cases": artifact.get("selected_cases", []),
        "summary": artifact.get("summary", {}),
    }


def _e(value: Any) -> str:
    return html.escape(str(value or ""))
