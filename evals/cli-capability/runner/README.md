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

## Validation

```sh
python3 evals/cli-capability/runner/validate.py
python3 -m unittest evals/cli-capability/runner/run_test.py
```
