# CLI Capability Verification-Failure Result

This is a sanitized, commit-ready result selected from a local live LLM run.
Raw traces, local paths, provider paths, shard directories, prompts, raw model
responses, and credentials are intentionally excluded.

## Source

- Source run id: `20260710-113525-kimi-k2-7-code-bdce0a`
- Mode: `live_llm`
- Experiment: `verification_failure`
- Level: `full`
- Model: `kimi-k2.7-code`
- Jobs: `8`

## Headline Metrics

- Verification gate: `10 / 10 = 100.00%`
- False promotions: `0`
- Generated candidates: `10`
- Failed probes: `10`
- Promoted bindings: `0`
- Negative evidence count: `10`

## Blocked Drift Cases

- `failure_gating:drift_archive_pack`: `verification_failed`
- `failure_gating:drift_archive_second_file`: `verification_failed`
- `failure_gating:drift_hash_sha256`: `verification_failed`
- `failure_gating:drift_hash_sha256_large`: `verification_failed`
- `failure_gating:drift_json_nested_drop`: `verification_failed`
- `failure_gating:drift_json_normalize`: `verification_failed`
- `failure_gating:drift_redact_multiline_tokens`: `verification_failed`
- `failure_gating:drift_redact_tokens`: `verification_failed`
- `failure_gating:drift_table_extract`: `verification_failed`
- `failure_gating:drift_table_name_extract`: `verification_failed`

## Files

- `metrics.json`: compact paper-facing metrics.
- `artifact.public.json`: sanitized machine-readable result.
- `report.html`: sanitized HTML report generated from `artifact.public.json`.
- `provenance.json`: non-secret reproduction metadata.

## Timing And Cost Signals

- Average verification-failure acquisition latency: `54.2 s`
- LLM calls: `40`
- Total proposal tokens: `81231`
