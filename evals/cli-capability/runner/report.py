from __future__ import annotations

import json
from pathlib import Path
from typing import Any

from constants import FLOW_MATRIX_STEPS, FLOW_SCHEMA_VERSION, STATUS_FAILED, STATUS_PASSED, STATUS_SKIPPED, SUITES
from util import escape, format_duration_value, format_duration_ms, fraction, strip_private_fields, write_json


class ArtifactWriter:
    def __init__(self, run_dir: Path, artifact: dict[str, Any]) -> None:
        self.run_dir = run_dir
        self.artifact = artifact
        self.summary_path = run_dir / "summary.json"
        self.flow_path = run_dir / "flow.json"
        self.html_path = run_dir / "index.html"
        self.artifact_path = run_dir / "artifact.json"

    def write(self) -> None:
        clean = strip_private_fields(self.artifact)
        write_json(self.summary_path, clean)
        write_json(self.flow_path, build_flow_artifact(clean))
        write_json(self.artifact_path, build_release_artifact(clean))
        self.html_path.write_text(render_html(clean), encoding="utf-8")


def build_flow_artifact(artifact: dict[str, Any]) -> dict[str, Any]:
    return {
        "schema_version": FLOW_SCHEMA_VERSION,
        "run": artifact.get("run") or {},
        "suites": {
            suite: {
                "cases": [
                    {
                        "id": case.get("id", ""),
                        "suite": case.get("suite", ""),
                        "intent": case.get("intent", ""),
                        "domain": case.get("domain", ""),
                        "providers": provider_flows(case),
                        "use": case.get("use") or [],
                        "baselines": case.get("baselines") or {},
                    }
                    for case in ((artifact.get("suites") or {}).get(suite) or {}).get("cases") or []
                ]
            }
            for suite in SUITES
        },
        "summary": artifact.get("summary") or {},
        "scores": artifact.get("scores") or {},
        "capability_model": artifact.get("capability_model") or {},
        "failure_taxonomy": artifact.get("failure_taxonomy") or [],
    }


def provider_flows(case: dict[str, Any]) -> list[dict[str, Any]]:
    return [
        {
            "provider": provider.get("id", ""),
            "command": provider.get("command", ""),
            "provider_path": provider.get("provider_path", ""),
            "provider_id": provider.get("provider_id", ""),
            "trace_id": provider.get("trace_id", ""),
            "status": provider_flow_status(provider),
            "steps": provider.get("steps") or [],
            "candidates": provider.get("candidates") or [],
        }
        for provider in case.get("providers") or []
    ]


def build_release_artifact(artifact: dict[str, Any]) -> dict[str, Any]:
    flows = build_flow_artifact(artifact)
    return {
        "schema_version": flows["schema_version"],
        "run": flows["run"],
        "summary": flows["summary"],
        "scores": flows["scores"],
        "capability_model": flows["capability_model"],
        "failure_taxonomy": flows["failure_taxonomy"],
        "trace_refs": trace_refs(flows),
    }


def trace_refs(flow: dict[str, Any]) -> list[dict[str, Any]]:
    rows = []
    for suite, suite_data in (flow.get("suites") or {}).items():
        for case in suite_data.get("cases") or []:
            for provider in case.get("providers") or []:
                trace_id = provider.get("trace_id", "")
                if trace_id:
                    rows.append({"suite": suite, "case_id": case.get("id", ""), "provider": provider.get("provider", ""), "trace_id": trace_id})
    return rows


def render_html(artifact: dict[str, Any]) -> str:
    flow = build_flow_artifact(artifact)
    return f"""<!doctype html>
<html>
<head>
<meta charset="utf-8">
<title>CLI Capability Benchmark</title>
<style>
body{{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;margin:24px;color:#222}}
.grid{{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:12px;margin:16px 0}}
.card{{border:1px solid #ddd;border-radius:8px;padding:12px;background:#fafafa}}
.primary .card{{background:#f6fbf7;border-color:#c9e4cf}}
.timing .card{{background:#f7f8fb}}
table{{border-collapse:collapse;width:100%;font-size:14px}}
td,th{{border:1px solid #ddd;padding:8px;vertical-align:top}}
th{{background:#f5f5f5;text-align:left}}
code{{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:12px}}
.section{{margin-top:28px}}
.step{{min-width:88px;text-align:center}}
.ok{{color:#126b31;font-weight:600}}
.fail{{color:#9b1c1c;font-weight:600}}
.muted{{color:#666}}
.warn{{color:#8a5a00;font-weight:600}}
summary{{cursor:pointer;font-weight:600;margin:12px 0}}
</style>
</head>
<body>
<h1>CLI Capability Benchmark</h1>
{render_overview(artifact)}
{render_main_result(artifact)}
{render_suite(flow, "acquisition", "Acquisition Suite")}
{render_suite(flow, "capability_model", "Capability Model Suite")}
{render_suite(flow, "reuse", "Reuse Suite")}
{render_capability_model(flow)}
{render_baselines(artifact)}
{render_timing(artifact)}
{render_negative_evidence(artifact)}
{render_raw_details(artifact)}
</body>
</html>
"""


def render_overview(artifact: dict[str, Any]) -> str:
    run = artifact.get("run") or {}
    llm = run.get("llm") or {}
    return f"""<section class="section">
<h2>Run Overview</h2>
<div class="grid">
{overview_card("Mode", run.get("mode", ""))}
{overview_card("Status", artifact.get("status", ""))}
{overview_card("Run ID", run.get("id", ""))}
{overview_card("Suites", ", ".join(run.get("selected_suites") or []))}
{overview_card("Cases", ", ".join(run.get("selected_cases") or []))}
{overview_card("Model", llm.get("model") or "n/a")}
{overview_card("Platform", f"{run.get('goos', '')} / {run.get('goarch', '')}")}
</div>
</section>"""


def render_main_result(artifact: dict[str, Any]) -> str:
    summary_root = artifact.get("summary") or {}
    total = summary_root.get("total") or {}
    reuse = (summary_root.get("suites") or {}).get("reuse") or {}
    scores = artifact.get("scores") or {}
    cards = [
        overview_card("Closed-loop success", format_metric("closed_loop_success_rate", scores.get("closed_loop_success_rate"))),
        overview_card("Intent use", fraction(reuse.get("use_oracle_pass_count"), reuse.get("held_out_uses"))),
        overview_card("Direct reuse", fraction(reuse.get("oracle_pass_count"), reuse.get("held_out_reuses"))),
        overview_card("Promoted bindings", total.get("promoted_bindings", 0)),
        overview_card("Provider yield", format_metric("provider_yield_rate", scores.get("provider_yield_rate"))),
        overview_card("Negative evidence", total.get("candidate_negative_evidence", 0)),
        overview_card("Failed closed loop", reuse.get("failed", 0)),
    ]
    return f"""<section class="section primary">
<h2>Main Result</h2>
<div class="grid">{''.join(cards)}</div>
</section>"""


def render_suite(flow: dict[str, Any], suite: str, title: str) -> str:
    rows = []
    for case in ((flow.get("suites") or {}).get(suite) or {}).get("cases") or []:
        for provider in case.get("providers") or []:
            rows.append(render_provider_row(case, provider))
    if not rows:
        rows.append(f"<tr><td colspan='{len(FLOW_MATRIX_STEPS) + 4}'><span class='muted'>none</span></td></tr>")
    headers = "".join(f"<th>{escape(short_step_label(name))}</th>" for name in FLOW_MATRIX_STEPS)
    return f"""<section class="section primary">
<h2>{escape(title)}</h2>
<table>
<tr><th>Case</th><th>Domain</th><th>Provider</th>{headers}<th>Final</th></tr>
{''.join(rows)}
</table>
</section>"""


def render_provider_row(case: dict[str, Any], provider: dict[str, Any]) -> str:
    steps = provider.get("steps") or []
    intent_steps = first_use_steps(case)
    cells = []
    for name in FLOW_MATRIX_STEPS:
        source = intent_steps if name.startswith("intent_use.") else steps
        cells.append(f"<td class='step'>{step_badge(find_step(source, name))}</td>")
    return (
        f"<tr><td><code>{escape(case.get('id', ''))}</code></td>"
        f"<td>{escape(case.get('domain', ''))}</td>"
        f"<td><code>{escape(provider.get('provider', ''))}</code><br>{escape(provider.get('provider_path', ''))}</td>"
        f"{''.join(cells)}<td>{status(provider.get('status'))}</td></tr>"
    )


def render_capability_model(flow: dict[str, Any]) -> str:
    model = flow.get("capability_model") or {}
    provider_rows = "".join(
        f"<tr><td><code>{escape(provider)}</code></td><td>{escape(', '.join(caps))}</td><td>{len(caps)}</td></tr>"
        for provider, caps in (model.get("providers") or {}).items()
    )
    capability_rows = "".join(
        f"<tr><td><code>{escape(capability)}</code></td><td>{escape(', '.join(providers))}</td><td>{len(providers)}</td></tr>"
        for capability, providers in (model.get("capabilities") or {}).items()
    )
    provider_rows = provider_rows or "<tr><td colspan='3'><span class='muted'>none</span></td></tr>"
    capability_rows = capability_rows or "<tr><td colspan='3'><span class='muted'>none</span></td></tr>"
    return f"""<section class="section">
<h2>Capability Model Evidence</h2>
<h3>Provider Coverage</h3>
<table><tr><th>Provider</th><th>Capabilities</th><th>Count</th></tr>{provider_rows}</table>
<h3>Capability Coverage</h3>
<table><tr><th>Capability</th><th>Providers</th><th>Binding Providers</th></tr>{capability_rows}</table>
</section>"""


def render_baselines(artifact: dict[str, Any]) -> str:
    baselines = ((artifact.get("summary") or {}).get("baselines") or {})
    rows = []
    for name, data in baselines.items():
        rows.append(
            f"<tr><td><code>{escape(name)}</code></td><td>{data.get('attempted', 0)}</td><td>{data.get('passed', 0)}</td>"
            f"<td>{format_metric('success_rate', data.get('success_rate'))}</td><td>{format_duration_value(data.get('avg_duration_ms'))}</td></tr>"
        )
    if not rows:
        rows.append("<tr><td colspan='5'><span class='muted'>none</span></td></tr>")
    return f"""<section class="section">
<h2>Reuse Baseline / Cost Amortization</h2>
<table><tr><th>Baseline</th><th>Attempted</th><th>Passed</th><th>Success</th><th>Avg latency</th></tr>{''.join(rows)}</table>
</section>"""


def render_timing(artifact: dict[str, Any]) -> str:
    scores = artifact.get("scores") or {}
    rows = [
        timing_row("Provider acquisition", scores.get("provider_acquisition_total_ms"), scores.get("provider_acquisition_avg_ms")),
        timing_row("Proposal LLM", scores.get("proposal_llm_total_ms"), scores.get("proposal_llm_avg_ms")),
        timing_row("Acquisition local overhead", scores.get("acquisition_local_overhead_total_ms"), None),
        timing_row("Intent use", scores.get("intent_use_total_ms"), scores.get("intent_use_avg_ms")),
        timing_row("Replay direct reuse", scores.get("replay_direct_reuse_total_ms"), scores.get("replay_direct_reuse_avg_ms")),
    ]
    return f"""<section class="section timing">
<h2>Workflow Timing</h2>
<table><tr><th>Flow</th><th>Total</th><th>Average</th></tr>{''.join(rows)}</table>
</section>"""


def render_negative_evidence(artifact: dict[str, Any]) -> str:
    rows = "".join(
        f"<tr><td>{escape(item.get('stage', ''))}</td><td>{escape(item.get('code', ''))}</td><td>{item.get('count', 0)}</td></tr>"
        for item in artifact.get("failure_taxonomy") or []
    )
    if not rows:
        rows = "<tr><td colspan='3'><span class='muted'>none</span></td></tr>"
    return f"""<section class="section">
<h2>Failure Taxonomy</h2>
<table><tr><th>Stage</th><th>Code</th><th>Count</th></tr>{rows}</table>
</section>"""


def render_raw_details(artifact: dict[str, Any]) -> str:
    return f"""<section class="section">
<details><summary>Raw scores</summary><pre>{escape(json.dumps(artifact.get("scores", {}), indent=2))}</pre></details>
<details><summary>Raw summary</summary><pre>{escape(json.dumps(artifact.get("summary", {}), indent=2))}</pre></details>
<details><summary>Capability model</summary><pre>{escape(json.dumps(artifact.get("capability_model", {}), indent=2))}</pre></details>
<details><summary>Failure taxonomy</summary><pre>{escape(json.dumps(artifact.get("failure_taxonomy", []), indent=2))}</pre></details>
</section>"""


def overview_card(label: str, value: Any) -> str:
    return f"<div class='card'>{escape(label)}<br><strong>{escape(value)}</strong></div>"


def first_use_steps(case: dict[str, Any]) -> list[dict[str, Any]]:
    uses = case.get("use") or []
    return (uses[0].get("steps") or []) if uses else []


def provider_flow_status(provider: dict[str, Any]) -> str:
    statuses = [step.get("status") for step in provider.get("steps") or []]
    if any(value == STATUS_FAILED for value in statuses):
        return STATUS_FAILED
    if statuses and all(value == STATUS_SKIPPED for value in statuses):
        return STATUS_SKIPPED
    if provider.get("status"):
        return provider["status"]
    return STATUS_PASSED if statuses else ""


def find_step(steps: list[dict[str, Any]], name: str) -> dict[str, Any] | None:
    matches = [step for step in steps if step.get("name") == name]
    if not matches:
        return None
    if any(step.get("status") == STATUS_FAILED for step in matches):
        return next(step for step in matches if step.get("status") == STATUS_FAILED)
    if any(step.get("status") == STATUS_PASSED for step in matches):
        return next(step for step in matches if step.get("status") == STATUS_PASSED)
    return matches[0]


def step_badge(step_value: dict[str, Any] | None) -> str:
    if not step_value:
        return status("")
    text = status(step_value.get("status"))
    duration = step_value.get("duration_ms")
    if duration is not None:
        text += f"<br><span class='muted'>{escape(format_duration_value(duration))}</span>"
    failure_value = step_value.get("failure") or {}
    if failure_value:
        text += f"<br><span class='fail'>{escape(failure_value.get('code', ''))}</span>"
    return text


def short_step_label(name: str) -> str:
    labels = {
        "provider.resolve": "Resolve",
        "provider.register": "Register",
        "acquisition.run": "Acquire",
        "acquisition.observe": "Observe",
        "proposal.surface": "Surface",
        "proposal.capability": "Capability",
        "proposal.binding": "Binding",
        "proposal.evidence": "Evidence",
        "acquisition.probe": "Probe",
        "acquisition.promote": "Promote",
        "direct_reuse.oracle": "Direct reuse",
        "intent_use.oracle": "Intent use",
    }
    return labels.get(name, name)


def timing_row(label: str, total: Any, average_value: Any) -> str:
    return f"<tr><td>{escape(label)}</td><td>{escape(format_duration_value(total))}</td><td>{escape(format_duration_value(average_value))}</td></tr>"


def status(value: Any) -> str:
    value = str(value or "")
    if value == STATUS_PASSED:
        return "<span class='ok'>passed</span>"
    if value == STATUS_SKIPPED:
        return "<span class='muted'>skipped</span>"
    if value:
        return f"<span class='fail'>{escape(value)}</span>"
    return "<span class='muted'>none</span>"


def format_metric(key: str, value: Any) -> str:
    if value is None:
        return "n/a"
    if isinstance(value, (int, float)) and (key.endswith("_rate") or key.endswith("_yield") or key == "success_rate"):
        return f"{value * 100:.1f}%"
    if isinstance(value, int) and key.endswith("_ms"):
        return format_duration_ms(value)
    return str(value)
