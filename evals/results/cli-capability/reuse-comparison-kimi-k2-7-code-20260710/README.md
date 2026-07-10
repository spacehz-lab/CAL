# CLI Capability Reuse Result

This is a sanitized, commit-ready result selected from a local benchmark run.
Raw traces, local paths, provider paths, shard directories, prompts, raw model
responses, and credentials are intentionally excluded.

## Source

- Source run id: `20260710-072146-kimi-k2-7-code-3956e3`
- Mode: `live_llm`
- Experiment: `repeated_reuse`
- Reuse profile: `comparison`
- Level: `full`
- Model: `kimi-k2.7-code`
- Jobs: `8`

## Headline Metrics

- Reuse gate: `10 / 10 = 100.00%`
- CAL use passed: `10 / 10`
- Promoted bindings available: `12`
- Average CAL use latency: `10.4 s`
- One-shot attempted: `10`
- One-shot passed: `7`
- One-shot total tokens: `17075`

## Files

- `metrics.json`: compact paper-facing metrics.
- `artifact.public.json`: sanitized machine-readable result.
- `report.html`: sanitized HTML report generated from `artifact.public.json`.
- `provenance.json`: non-secret reproduction metadata.
