from __future__ import annotations

import json
from pathlib import Path
from typing import Any

from constants import (
    BASELINE_LLM_ONESHOT,
    EXPERIMENT_ACQUISITION,
    EXPERIMENT_CAPABILITY_STRUCTURE,
    EXPERIMENT_REPEATED_REUSE,
    EXPERIMENT_VERIFICATION_FAILURE,
    FLOW_SCHEMA_VERSION,
)
from util import escape, format_duration_value, fraction, ratio, strip_private_fields, write_json


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
        "cases": [
            {
                "id": case.get("id", ""),
                "case_key": case.get("case_key", ""),
                "scenario_group": case.get("scenario_group", ""),
                "paper_experiments": case.get("paper_experiments") or [],
                "provider_class": case.get("provider_class", ""),
                "acquisition_mode": case.get("acquisition_mode", ""),
                "failure_type": case.get("failure_type", ""),
                "intent": case.get("intent", ""),
                "domain": case.get("domain", ""),
                "providers": provider_flows(case),
                "use": case.get("use") or [],
                "baselines": case.get("baselines") or {},
            }
            for case in artifact.get("cases") or []
        ],
        "summary": artifact.get("summary") or {},
        "scores": artifact.get("scores") or {},
        "experiment_gates": artifact.get("experiment_gates") or {},
        "coverage": artifact.get("coverage") or {},
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
            "status": provider.get("status", ""),
            "llm_call_count": provider.get("llm_call_count", 0),
            "llm_duration_ms": provider.get("llm_duration_ms", 0),
            "prompt_tokens": provider.get("prompt_tokens", 0),
            "completion_tokens": provider.get("completion_tokens", 0),
            "total_tokens": provider.get("total_tokens", 0),
            "steps": provider.get("steps") or [],
            "candidates": provider.get("candidates") or [],
        }
        for provider in case.get("providers") or []
    ]


def build_release_artifact(artifact: dict[str, Any]) -> dict[str, Any]:
    flow = build_flow_artifact(artifact)
    return {
        "schema_version": flow["schema_version"],
        "run": flow["run"],
        "summary": flow["summary"],
        "scores": flow["scores"],
        "experiment_gates": flow["experiment_gates"],
        "coverage": flow["coverage"],
        "capability_model": flow["capability_model"],
        "failure_taxonomy": flow["failure_taxonomy"],
        "trace_refs": trace_refs(flow),
    }


def trace_refs(flow: dict[str, Any]) -> list[dict[str, Any]]:
    rows = []
    for case in flow.get("cases") or []:
        for provider in case.get("providers") or []:
            trace_id = provider.get("trace_id", "")
            if trace_id:
                rows.append(
                    {
                        "case_key": case.get("case_key", ""),
                        "case_id": case.get("id", ""),
                        "provider": provider.get("provider", ""),
                        "trace_id": trace_id,
                    }
                )
    return rows


def render_html(artifact: dict[str, Any]) -> str:
    flow = build_flow_artifact(artifact)
    return f"""<!doctype html>
<html>
<head>
<meta charset="utf-8">
<title>CLI Capability Paper Benchmark</title>
<style>
body{{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;margin:24px;color:#222}}
.grid{{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:12px;margin:16px 0}}
.card{{border:1px solid #ddd;border-radius:8px;padding:12px;background:#fafafa}}
.primary .card{{background:#f6fbf7;border-color:#c9e4cf}}
table{{border-collapse:collapse;width:100%;font-size:14px;margin:12px 0}}
td,th{{border:1px solid #ddd;padding:8px;vertical-align:top}}
th{{background:#f5f5f5;text-align:left}}
code{{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:12px}}
.section{{margin-top:28px}}
.ok{{color:#126b31;font-weight:600}}
.fail{{color:#9b1c1c;font-weight:600}}
.muted{{color:#666}}
summary{{cursor:pointer;font-weight:600;margin:12px 0}}
</style>
</head>
<body>
<h1>CLI Capability Paper Benchmark</h1>
{render_overview(artifact)}
{render_acquisition_section(flow)}
{render_failure_section(flow)}
{render_capability_structure_section(flow)}
{render_repeated_reuse_section(flow)}
{render_raw_details(artifact)}
</body>
</html>
"""


def render_overview(artifact: dict[str, Any]) -> str:
    run = artifact.get("run") or {}
    llm = run.get("llm") or {}
    coverage = artifact.get("coverage") or {}
    selected_experiments = run.get("selected_experiments") or []
    selected_cases = run.get("selected_cases") or []
    cards = [
        overview_card("Mode", run.get("mode", "")),
        overview_card("Status", artifact.get("status", "")),
        overview_card("Run ID", run.get("id", "")),
        overview_card("Experiments", ", ".join(selected_experiments)),
        overview_card("Cases", f"{len(selected_cases)} selected / {coverage.get('distinct_case_count', 0)} distinct"),
        overview_card("Jobs", run.get("jobs", 1)),
        overview_card("Model", llm.get("model") or "n/a"),
        overview_card("Platform", f"{run.get('goos', '')} / {run.get('goarch', '')}"),
    ]
    return f"""<section class="section">
<h2>Run Overview</h2>
<div class="grid">{''.join(cards)}</div>
{render_selected_cases(selected_experiments, selected_cases)}
</section>"""


def render_acquisition_section(flow: dict[str, Any]) -> str:
    return f"""<section class="section primary">
<h2>Experiment 1: Acquiring Capabilities From Provider Surfaces</h2>
{render_gate(flow, EXPERIMENT_ACQUISITION)}
{render_acquisition_table(flow)}
</section>"""


def render_failure_section(flow: dict[str, Any]) -> str:
    return f"""<section class="section">
<h2>Experiment 2: Verification And Failure Gating</h2>
{render_gate(flow, EXPERIMENT_VERIFICATION_FAILURE)}
{render_failure_table(flow)}
</section>"""


def render_capability_structure_section(flow: dict[str, Any]) -> str:
    return f"""<section class="section">
<h2>Experiment 3: Capability Structure Evidence</h2>
{render_gate(flow, EXPERIMENT_CAPABILITY_STRUCTURE)}
{render_capability_model_tables(flow)}
</section>"""


def render_repeated_reuse_section(flow: dict[str, Any]) -> str:
    return f"""<section class="section primary">
<h2>Experiment 4: Repeated Held-Out Reuse</h2>
{render_gate(flow, EXPERIMENT_REPEATED_REUSE)}
{render_reuse_summary(flow)}
{render_reuse_table(flow)}
{render_reuse_comparison(flow)}
</section>"""


def render_gate(flow: dict[str, Any], experiment: str) -> str:
    gate_value = ((flow.get("experiment_gates") or {}).get(experiment) or {})
    if not gate_value:
        return ""
    status_label = "passed" if gate_value.get("passed") else "failed"
    cards = [
        overview_card("Gate", status_label),
        overview_card("Actual", f"{gate_value.get('numerator', 0)} / {gate_value.get('denominator', 0)} = {format_percent(gate_value.get('actual'))}"),
        overview_card("Target", f">= {format_percent(gate_value.get('target'))}"),
    ]
    if "false_promotions" in gate_value:
        cards.append(overview_card("False promotions", gate_value.get("false_promotions", 0)))
    if gate_value.get("skipped"):
        cards.append(overview_card("Skipped", gate_value.get("skipped", 0)))
    return f"<div class='grid'>{''.join(cards)}</div>"


def render_acquisition_table(flow: dict[str, Any]) -> str:
    rows = []
    for case in experiment_cases(flow, EXPERIMENT_ACQUISITION):
        for provider in case.get("providers") or []:
            candidates = provider.get("candidates") or []
            rows.append(
                "<tr>"
                f"<td><code>{escape(case.get('case_key', ''))}</code></td>"
                f"<td>{escape(case.get('provider_class', ''))}</td>"
                f"<td>{escape(case.get('acquisition_mode', ''))}</td>"
                f"<td><code>{escape(provider.get('provider', ''))}</code></td>"
                f"<td>{len(candidates)}</td>"
                f"<td>{count_probes(candidates)}</td>"
                f"<td>{count_verified(candidates)}</td>"
                f"<td>{count_promoted(candidates)}</td>"
                f"<td>{format_duration_value(provider.get('acquisition_duration_ms'))} / {escape(provider.get('total_tokens', 0))}</td>"
                f"<td><code>{escape(provider.get('trace_id', ''))}</code></td>"
                "</tr>"
            )
    return table_or_empty(
        "<tr><th>Case</th><th>Provider class</th><th>Mode</th><th>Provider</th><th>Candidates</th><th>Probes</th><th>Verified</th><th>Promoted</th><th>Latency / tokens</th><th>Trace</th></tr>",
        rows,
    )


def render_failure_table(flow: dict[str, Any]) -> str:
    rows = []
    for case in experiment_cases(flow, EXPERIMENT_VERIFICATION_FAILURE):
        for provider in case.get("providers") or []:
            candidates = provider.get("candidates") or []
            promoted = count_promoted(candidates)
            rows.append(
                "<tr>"
                f"<td><code>{escape(case.get('case_key', ''))}</code></td>"
                f"<td>{escape(case.get('failure_type', ''))}</td>"
                f"<td>{'yes' if candidates else 'no'}</td>"
                f"<td>{count_failed_probes(candidates)} failed / {count_verified(candidates)} passed</td>"
                f"<td>{'promoted' if promoted else 'blocked'}</td>"
                f"<td>{'true' if promoted else 'false'}</td>"
                f"<td><code>{escape(provider.get('trace_id', ''))}</code></td>"
                "</tr>"
            )
    return table_or_empty(
        "<tr><th>Case</th><th>Failure type</th><th>Candidate generated</th><th>Probe/verifier</th><th>Promotion</th><th>False promotion</th><th>Evidence</th></tr>",
        rows,
    )


def render_capability_model_tables(flow: dict[str, Any]) -> str:
    model = flow.get("capability_model") or {}
    provider_rows = [
        f"<tr><td><code>{escape(provider)}</code></td><td>{escape(', '.join(caps))}</td><td>{len(caps)}</td></tr>"
        for provider, caps in (model.get("providers") or {}).items()
    ]
    capability_rows = [
        f"<tr><td><code>{escape(capability)}</code></td><td>{escape(', '.join(providers))}</td><td>{len(providers)}</td></tr>"
        for capability, providers in (model.get("capabilities") or {}).items()
    ]
    check_rows = [
        f"<tr><td><code>{escape(check.get('case_key') or check.get('case_id', ''))}</code></td><td>{escape(check.get('expectation', ''))}</td>"
        f"<td>{check.get('actual', 0)} / {check.get('expected', 0)}</td><td>{status(check.get('status'))}</td></tr>"
        for check in model.get("checks") or []
    ]
    return (
        "<h3>Provider -> Capability Map</h3>"
        + table_or_empty("<tr><th>Provider</th><th>Capabilities</th><th>Count</th></tr>", provider_rows)
        + "<h3>Capability -> Provider/Binding Map</h3>"
        + table_or_empty("<tr><th>Capability</th><th>Providers</th><th>Binding providers</th></tr>", capability_rows)
        + "<h3>Structure Checks</h3>"
        + table_or_empty("<tr><th>Case</th><th>Expectation</th><th>Actual / Expected</th><th>Status</th></tr>", check_rows)
    )


def render_reuse_summary(flow: dict[str, Any]) -> str:
    reuse = ((flow.get("summary") or {}).get("experiments") or {}).get(EXPERIMENT_REPEATED_REUSE) or {}
    cards = [
        overview_card("End-to-end reuse", fraction(reuse.get("use_oracle_pass_count"), reuse.get("held_out_uses"))),
        overview_card("Conditional reuse", fraction(reuse.get("conditional_use_oracle_pass_count"), reuse.get("eligible_held_out_uses"))),
        overview_card("Avg CAL use latency", format_duration_value(reuse.get("avg_use_ms"))),
        overview_card("Upstream acquisition failures", reuse.get("upstream_acquisition_failure_count", 0)),
    ]
    return f"<div class='grid'>{''.join(cards)}</div>"


def render_reuse_table(flow: dict[str, Any]) -> str:
    rows = []
    for case in experiment_cases(flow, EXPERIMENT_REPEATED_REUSE):
        for use in case.get("use") or []:
            rows.append(
                "<tr>"
                f"<td><code>{escape(case.get('case_key', ''))}</code></td>"
                f"<td><code>{escape(use.get('fixture_id', ''))}</code></td>"
                f"<td>{escape(case.get('provider_class', ''))}</td>"
                f"<td>{status(use.get('status'))}</td>"
                f"<td><code>{escape((use.get('selection') or {}).get('binding_id', ''))}</code></td>"
                f"<td>{format_duration_value(use.get('duration_ms'))}</td>"
                "</tr>"
            )
    return table_or_empty("<tr><th>Case</th><th>Round</th><th>Provider class</th><th>CAL result</th><th>Binding</th><th>Latency</th></tr>", rows)


def render_reuse_comparison(flow: dict[str, Any]) -> str:
    reuse = ((flow.get("summary") or {}).get("experiments") or {}).get(EXPERIMENT_REPEATED_REUSE) or {}
    baselines = ((flow.get("summary") or {}).get("baselines") or {})
    rows = [
        method_row("CAL use", reuse.get("held_out_uses", 0), reuse.get("use_oracle_pass_count", 0), reuse.get("oracle_use_success_rate", 0), reuse.get("avg_use_ms", 0), reuse.get("run_stage_llm_calls", 0), reuse.get("total_tokens", 0), "yes")
    ]
    one_shot = baselines.get(BASELINE_LLM_ONESHOT)
    if one_shot:
        rows.append(
            method_row(
                BASELINE_LLM_ONESHOT,
                one_shot.get("attempted", 0),
                one_shot.get("passed", 0),
                one_shot.get("success_rate", 0),
                one_shot.get("avg_duration_ms", 0),
                one_shot.get("llm_calls", 0),
                one_shot.get("total_tokens", 0),
                "no",
            )
        )
    return "<h3>Repeated Reuse Method Comparison</h3>" + table_or_empty(
        "<tr><th>Method</th><th>Attempted</th><th>Passed</th><th>Success</th><th>Avg latency</th><th>LLM calls</th><th>Tokens</th><th>Reusable binding</th></tr>",
        rows,
    )


def method_row(name: str, attempted: Any, passed: Any, success_rate: Any, avg_latency_ms: Any, llm_calls: Any, tokens: Any, reusable: Any) -> str:
    return (
        f"<tr><td><code>{escape(name)}</code></td><td>{escape(attempted)}</td><td>{escape(passed)}</td>"
        f"<td>{format_percent(success_rate)}</td><td>{format_duration_value(avg_latency_ms)}</td>"
        f"<td>{escape(llm_calls)}</td><td>{escape(tokens)}</td><td>{escape(reusable)}</td></tr>"
    )


def render_raw_details(artifact: dict[str, Any]) -> str:
    return f"""<section class="section">
<details><summary>Raw scores</summary><pre>{escape(json.dumps(artifact.get("scores", {}), indent=2))}</pre></details>
<details><summary>Raw summary</summary><pre>{escape(json.dumps(artifact.get("summary", {}), indent=2))}</pre></details>
<details><summary>Capability model</summary><pre>{escape(json.dumps(artifact.get("capability_model", {}), indent=2))}</pre></details>
<details><summary>Failure taxonomy</summary><pre>{escape(json.dumps(artifact.get("failure_taxonomy", []), indent=2))}</pre></details>
</section>"""


def experiment_cases(flow: dict[str, Any], experiment: str) -> list[dict[str, Any]]:
    return [case for case in flow.get("cases") or [] if experiment in (case.get("paper_experiments") or [])]


def count_probes(candidates: list[dict[str, Any]]) -> int:
    return sum(1 for candidate in candidates if (candidate.get("probe") or {}).get("status") not in ("", "not_run"))


def count_verified(candidates: list[dict[str, Any]]) -> int:
    return sum(1 for candidate in candidates if (candidate.get("probe") or {}).get("passed"))


def count_failed_probes(candidates: list[dict[str, Any]]) -> int:
    return sum(1 for candidate in candidates if (candidate.get("probe") or {}).get("status") == "failed")


def count_promoted(candidates: list[dict[str, Any]]) -> int:
    return sum(1 for candidate in candidates if (candidate.get("promotion") or {}).get("binding_id"))


def table_or_empty(header: str, rows: list[str]) -> str:
    body = "".join(rows) if rows else "<tr><td colspan='12'><span class='muted'>none</span></td></tr>"
    return f"<table>{header}{body}</table>"


def overview_card(label: str, value: Any) -> str:
    return f"<div class='card'>{escape(label)}<br><strong>{escape(value)}</strong></div>"


def render_selected_cases(selected_experiments: list[str], selected_cases: list[str]) -> str:
    experiment_text = ", ".join(selected_experiments) or "none"
    case_items = "".join(f"<li><code>{escape(item)}</code></li>" for item in selected_cases)
    if not case_items:
        case_items = "<li><span class='muted'>none</span></li>"
    return f"""<details>
<summary>Selected experiments and cases</summary>
<p><strong>Experiments:</strong> {escape(experiment_text)}</p>
<ul>{case_items}</ul>
</details>"""


def status(value: Any) -> str:
    text = str(value or "")
    if text == "passed":
        return f"<span class='ok'>{escape(text)}</span>"
    if text == "failed":
        return f"<span class='fail'>{escape(text)}</span>"
    return escape(text)


def format_percent(value: Any) -> str:
    try:
        return f"{float(value) * 100:.1f}%"
    except (TypeError, ValueError):
        return "0.0%"
