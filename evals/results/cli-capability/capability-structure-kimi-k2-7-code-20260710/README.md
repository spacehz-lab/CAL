# CLI Capability Structure Result

This is a sanitized, commit-ready result selected from a local live LLM run.
Raw traces, local paths, provider paths, shard directories, prompts, raw model
responses, and credentials are intentionally excluded.

## Source

- Source run id: `20260710-084318-kimi-k2-7-code-d45a32`
- Mode: `live_llm`
- Experiment: `capability_structure`
- Level: `full`
- Model: `kimi-k2.7-code`
- Jobs: `8`

## Headline Metrics

- Capability-structure gate: `10 / 10 = 100.00%`
- Structure checks: `10` passed, `0` failed, `0` skipped
- Acquisition support gate: `14 / 14 = 100.00%`
- Multi-capability providers: `3`
- Multi-binding capabilities: `5`
- Provider records: `10`
- Capability records: `7`

## Files

- `metrics.json`: compact paper-facing metrics.
- `artifact.public.json`: sanitized machine-readable result.
- `report.html`: sanitized HTML report generated from `artifact.public.json`.
- `provenance.json`: non-secret reproduction metadata.

## Timing And Cost Signals

- Average acquisition latency: `3.4 min`
- LLM calls: `56`
- Total proposal tokens: `254043`
