# CLI Capability Results

This directory is for compact, commit-ready summaries selected from local eval
runs.

Do not copy raw `CAL_HOME`, API keys, raw prompts, full LLM responses, full
provider output dumps, or machine-specific directories here.

Local generated runs belong under `evals/out/cli-capability/`, which is ignored
by git and may be deleted at any time.

Use the exporter to curate a selected local run:

```sh
python3 evals/cli-capability/runner/export_result.py \
  --run evals/out/cli-capability/<run-id> \
  --name <paper-result-name>
```

The exporter writes `artifact.public.json`, `metrics.json`, `provenance.json`,
`README.md`, and `report.html`, then fails if the public directory still
contains local paths, raw model responses, trace ids, provider paths, shard
paths, or secret-like values.

## Committed Result Bundles

The current checked-in result bundles are sanitized summaries from Kimi
`kimi-k2.7-code` live LLM runs on 2026-07-10.

| Directory | Experiment | Headline |
| --- | --- | --- |
| `acquisition-full-kimi-k2-7-code-20260710` | `acquisition` | Acquisition gate `21 / 22`. |
| `verification-failure-kimi-k2-7-code-20260710` | `verification_failure` | Invalid candidates blocked `5 / 5`. |
| `verification-failure-expanded-kimi-k2-7-code-20260710` | `verification_failure` expanded profile | Invalid candidates blocked `10 / 10`; false promotions `0`. |
| `reuse-effectiveness-kimi-k2-7-code-20260710` | `repeated_reuse` effectiveness profile | CAL reuse passed `17 / 17`. |
| `reuse-comparison-kimi-k2-7-code-20260710` | `repeated_reuse` comparison profile | CAL reuse passed `10 / 10`; LLM one-shot passed `7 / 10`. |
| `reuse-comparison-expanded-kimi-k2-7-code-20260710` | `repeated_reuse` expanded comparison profile | CAL reuse passed `30 / 30`; LLM one-shot passed `21 / 30`. |
| `capability-structure-kimi-k2-7-code-20260710` | `capability_structure` | Structure checks passed `10 / 10`; acquisition support gate `14 / 14`. |

Each bundle has its own `README.md` and `metrics.json` with the exact run id,
model, gate metrics, and cost/timing counters.
