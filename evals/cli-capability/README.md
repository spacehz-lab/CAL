# CLI Capability Paper Benchmark

This directory defines the CAL arXiv v0 paper benchmark surface.

Benchmark id: `cli-capability`
Benchmark version: `paper-v0`

The benchmark is scenario-first. It is designed to produce trace-backed tables
for the paper experiments:

```text
Experiment 1: Acquiring Capabilities From Provider Surfaces
Experiment 2: Verification And Failure Gating
Experiment 3: Repeated Held-Out Reuse
Experiment 4: Capability Structure Evidence
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
  tools/bin/
  runner/
  report/
```

`scenarios/*.jsonl` is the benchmark case source of truth.
`tools/bin/` contains eval-local synthetic enterprise CLIs used for uncommon
provider cases; the runner prepends this directory to worker `PATH`.

Current scenario surface:

- `acquisition`: 16 cases, including 4 uncommon provider-suite full-acquisition cases.
- `verification_failure`: 5 synthetic provider-drift cases.
- `repeated_reuse`: 17 cases with held-out reuse rounds.
- `capability_structure`: structure checks attached to acquisition cases.

The uncommon provider slice uses `acmejson`, `corp-redact`, `datapick`, and
`packnote`. These are intentionally small eval-local command surfaces whose
syntax is not a standard Unix interface. Intent-guided acquisition still uses a
task hint; full acquisition omits the task hint by default and checks whether a
provider-wide run promotes the expected command surfaces for that provider.

The verification-failure slice uses eval-local drift providers whose help text
advertises normal capabilities while their implementations return semantically
wrong outputs. A correct run may generate candidates, but deterministic probes
must block promotion.

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

Held-out reuse uses `reuse.rounds`, not `reuse.fixtures`. Reuse reporting has
two paper profiles:

- `--reuse-profile effectiveness`: 17 cases, first held-out round only, no
  one-shot baseline.
- `--reuse-profile comparison`: 8 cases tagged `reuse_comparison`, 10 held-out
  rounds total, one `llm_oneshot` baseline attempt per round using the case's
  configured `baseline_provider`.

Full-acquisition cases may declare `expected_capabilities`. The summary and
HTML report use those expected command surfaces to compute discovery coverage
from promoted bindings. The report renders intent-guided and full-acquisition
runs as separate tables.

## Dependency Model

The benchmark is a fixed-sample DAG:

```text
Fixed Scenario Matrix
  -> acquisition/failure execution
  -> promoted and negative records
  -> capability-structure analysis
  -> repeated held-out reuse
```

`repeated_reuse` is a pure reuse experiment by default. It seeds the shard with
verified replay capability records, then calls `calctl use` with only the user
intent and runtime inputs. It does not pass a provider id, capability id, or
binding id to the use step. The use call passes `--strategy best`, so live LLM
runs use bounded LLM-assisted selection over the local promoted binding
shortlist. Use `--reuse-seed self` only when you want the older end-to-end
diagnostic mode where each reuse shard first runs acquisition.

Seeded reuse supports single-capability multi-binding cases: replay records for
the same capability are merged into one capability file with multiple promoted
bindings, such as the hash, search, JSON, and archive cases.

Reuse reports both:

- end-to-end reuse rate over all planned reuse rounds
- conditional reuse rate over rounds whose case produced a promoted binding

The effectiveness profile is the broad reuse-validity table. The comparison
profile is the focused CAL reuse vs LLM one-shot table.

`--experiment` also controls the experiment labels written to the artifact and
HTML report. Running one group, such as `--experiment capability_structure`,
produces a single-group report rather than also counting overlapping
acquisition labels.

## Parallel Execution

The runner supports case-level parallelism:

```sh
python3 evals/cli-capability/runner/run.py \
  --mode replay \
  --experiment acquisition,repeated_reuse \
  --level focus \
  --jobs 4 \
  --reuse-seed replay
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
