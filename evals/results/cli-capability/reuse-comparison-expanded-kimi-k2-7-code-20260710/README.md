# CLI Capability Reuse Result

This is a sanitized, commit-ready result selected from a local benchmark run.
Raw traces, local paths, provider paths, shard directories, prompts, raw model
responses, and credentials are intentionally excluded.

## Source

- Source run id: `20260710-101711-kimi-k2-7-code-135910`
- Mode: `live_llm`
- Experiment: `repeated_reuse`
- Reuse profile: `comparison`
- Level: `full`
- Model: `kimi-k2.7-code`
- Jobs: `8`

## Headline Metrics

- Reuse gate: `30 / 30 = 100.00%`
- CAL use passed: `30 / 30`
- Promoted bindings available: `20`
- Average CAL use latency: `7.3 s`
- One-shot attempted: `30`
- One-shot passed: `21`
- One-shot total tokens: `30162`

## Files

- `metrics.json`: compact paper-facing metrics.
- `artifact.public.json`: sanitized machine-readable result.
- `report.html`: sanitized HTML report generated from `artifact.public.json`.
- `provenance.json`: non-secret reproduction metadata.
