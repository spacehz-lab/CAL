# CLI Capability Acquisition Result

This is a sanitized, commit-ready result selected from a local live LLM run.
Raw traces, local paths, provider paths, shard directories, prompts, raw model
responses, and credentials are intentionally excluded.

## Source

- Source run id: `20260710-054743-kimi-k2-7-code-5b0eea`
- Mode: `live_llm`
- Experiment: `acquisition`
- Level: `full`
- Model: `kimi-k2.7-code`
- Jobs: `8`

## Headline Metrics

- Acquisition gate: `21 / 22 = 95.45%`
- Intent-guided providers with promoted bindings: `17 / 18`
- Full-acquisition providers with promoted bindings: `4 / 4`
- Full discovery coverage: `7 / 8 = 87.5%`
- Multi-cap promoted provider suites: `3 / 4 = 75.0%`

## Known Gaps

- `acquisition:enterprise_packnote_suite` missing: manifest

## Files

- `metrics.json`: compact paper-facing metrics.
- `artifact.public.json`: sanitized machine-readable result.
- `report.html`: sanitized HTML report generated from `artifact.public.json`.
- `provenance.json`: non-secret reproduction metadata.

## Timing And Cost Signals

- Average intent-guided acquisition latency: `2.7 min`
- Average full-acquisition latency: `1.8 min`
- Total proposal tokens: `370717`
