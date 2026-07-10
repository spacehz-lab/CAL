# CLI Capability Reuse Result

This is a sanitized, commit-ready result selected from a local benchmark run.
Raw traces, local paths, provider paths, shard directories, prompts, raw model
responses, and credentials are intentionally excluded.

## Source

- Source run id: `20260710-071823-kimi-k2-7-code-d569b1`
- Mode: `live_llm`
- Experiment: `repeated_reuse`
- Reuse profile: `effectiveness`
- Level: `full`
- Model: `kimi-k2.7-code`
- Jobs: `8`

## Headline Metrics

- Reuse gate: `17 / 17 = 100.00%`
- CAL use passed: `17 / 17`
- Promoted bindings available: `23`
- Average CAL use latency: `8.8 s`
- One-shot attempted: `0`
- One-shot passed: `0`
- One-shot total tokens: `0`

## Files

- `metrics.json`: compact paper-facing metrics.
- `artifact.public.json`: sanitized machine-readable result.
- `report.html`: sanitized HTML report generated from `artifact.public.json`.
- `provenance.json`: non-secret reproduction metadata.
