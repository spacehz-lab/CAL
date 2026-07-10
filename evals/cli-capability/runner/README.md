# CLI Capability Paper Runner

The runner executes the scenario-first CAL paper benchmark. It loads
`scenarios/*.jsonl`.

## Basic Commands

Replay run:

```sh
python3 evals/cli-capability/runner/run.py \
  --mode replay \
  --experiment acquisition,repeated_reuse \
  --level focus \
  --jobs 4
```

Live LLM run:

```sh
python3 evals/cli-capability/runner/run.py \
  --mode live_llm \
  --experiment acquisition,repeated_reuse \
  --level focus \
  --jobs 4 \
  --llm-jobs 2
```

Targeted run:

```sh
python3 evals/cli-capability/runner/run.py \
  --mode replay \
  --experiment repeated_reuse \
  --case repeated_reuse:file_hash_sha1
```

Pure reuse is the default. `repeated_reuse` seeds verified replay capability
records into the shard, then runs `calctl use` with only the user intent and
inputs. It does not pin provider, capability, or binding ids. The use command
passes `--strategy best`, so live runs use LLM-assisted selection over the
seeded or acquired local shortlist. Use `--reuse-seed self` for the older
end-to-end diagnostic mode that reacquires in each reuse shard.

Paper reuse uses two profiles:

- `--reuse-profile effectiveness`: all 17 reuse cases, one held-out round per
  case, no baseline. This is the broad reuse-validity table.
- `--reuse-profile comparison`: 14 cases tagged `reuse_comparison`, 30 held-out
  rounds total, and one `llm_oneshot` baseline attempt per round using the
  case's configured `baseline_provider`.

Seeded records are merged by capability id, so cases with two providers for the
same capability become one capability with multiple promoted bindings.

Single-group runs write single-group artifacts and HTML. For example,
`--experiment capability_structure` keeps only the capability-structure label in
the run output and seeds records when acquisition is not selected.

Acquisition reports split intent-guided and full-acquisition runs into separate
tables. Full-acquisition cases use provider-suite scenarios with
`expected_capabilities`, so the report also includes discovery coverage over the
expected promoted command surfaces.

## Selection Flags

- `--experiment`: comma-separated paper experiments:
  `acquisition`, `verification_failure`, `capability_structure`,
  `repeated_reuse`
- `--case`: comma-separated case ids or `scenario_group:case_id` keys
- `--provider-class`: filter by paper provider class
- `--tag`: filter by scenario tag
- `--failure-type`: filter controlled failure cases
- `--jobs`: case-level worker count
- `--llm-jobs`: maximum live LLM worker count
- `--reuse-seed`: `replay` for seeded pure reuse, or `self` for acquisition
  inside each reuse shard
- `--reuse-profile`: `all`, `effectiveness`, or `comparison`

## Execution Model

Each case worker gets an isolated shard:

```text
<run-dir>/shards/<scenario-case-key>/
  home/
  cald.log
```

The worker starts its own `cald` with `CAL_HOME` pointing to that shard. The
main process aggregates case artifacts into:

- `summary.json`
- `flow.json`
- `artifact.json`
- `index.html`

This design makes acquisition parallel without sharing provider registration,
promotion, trace, or output state across cases.

The runner also prepends `evals/cli-capability/tools/bin` to each worker `PATH`.
That directory contains eval-local synthetic enterprise CLIs for the uncommon
provider slice and is intentionally scoped to benchmark execution.

## Validation

```sh
python3 evals/cli-capability/runner/validate.py
python3 -m unittest evals/cli-capability/runner/run_test.py
```
