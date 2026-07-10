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
