# CLI Capability Paper Benchmark

This directory defines the CAL arXiv v0 paper benchmark surface.

Benchmark id: `cli-capability`
Benchmark version: `paper-v0`

The benchmark is scenario-first. It is designed to produce trace-backed tables
for the paper experiments:

```text
Experiment 1: Acquiring Capabilities From Provider Surfaces
Experiment 2: Verification And Failure Gating
Experiment 3: Capability Structure Evidence
Experiment 4: Repeated Held-Out Reuse
```

## Layout

```text
evals/cli-capability/
  scenarios/
    acquisition.jsonl
    failure_gating.jsonl
    capability_structure.jsonl
    repeated_reuse.jsonl
  providers.json
  fixtures/
  oracles/
  baselines/
  runner/
  report/
```

`scenarios/*.jsonl` is the benchmark case source of truth.

## Case Contract

Each scenario case declares the paper experiments it contributes to:

```json
{
  "id": "json_query_jq_known_cli",
  "level": "focus",
  "domain": "structured_data",
  "provider_class": "known_cli",
  "provider_candidates": ["jq", "python3"],
  "scenario_tags": ["known_cli", "intent_guided", "multi_binding"],
  "paper_experiments": ["acquisition", "capability_structure", "repeated_reuse"],
  "acquisition_mode": "intent_guided",
  "failure_type": "",
  "intent": "Query a JSON field and write the result.",
  "acquisition": {"fixtures": []},
  "reuse": {"rounds": []},
  "oracle": {}
}
```

Held-out reuse uses `reuse.rounds`, not `reuse.fixtures`. Repeated one-shot
baselines run once per held-out round.

## Dependency Model

The benchmark is a fixed-sample DAG:

```text
Fixed Scenario Matrix
  -> acquisition/failure execution
  -> promoted and negative records
  -> capability-structure analysis
  -> repeated held-out reuse
```

Later experiments may consume records produced by earlier stages, but they do
not select cases after seeing acquisition success. Reuse reports both:

- end-to-end reuse rate over all planned reuse rounds
- conditional reuse rate over rounds whose case produced a promoted binding

## Parallel Execution

The runner supports case-level parallelism:

```sh
python3 evals/cli-capability/runner/run.py \
  --mode replay \
  --experiment acquisition,repeated_reuse \
  --level focus \
  --jobs 4
```

Each worker uses an isolated `CAL_HOME` under the run directory and starts its
own `cald` instance. The main process aggregates shard artifacts and writes the
paper report.

For live LLM runs, use a smaller worker count:

```sh
python3 evals/cli-capability/runner/run.py \
  --mode live_llm \
  --experiment acquisition,repeated_reuse \
  --level focus \
  --jobs 4 \
  --llm-jobs 2
```

`--llm-jobs` limits live model concurrency to avoid turning the benchmark into
an API rate-limit test.

## Outputs

Generated results are written under `evals/out/cli-capability/<run-id>/` by
default:

- `summary.json`: full sanitized artifact
- `flow.json`: flow-shaped artifact for trace inspection
- `artifact.json`: release-oriented summary with trace refs
- `index.html`: paper experiment report
- `shards/`: per-case isolated homes, daemon logs, and traces

Generated outputs are not committed by default. A paper release should commit
only sanitized trace excerpts or exact release artifacts with secrets and
machine-local paths removed where possible.

## Validation

```sh
python3 evals/cli-capability/runner/validate.py
python3 -m unittest evals/cli-capability/runner/run_test.py
```
