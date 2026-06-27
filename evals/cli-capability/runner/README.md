# CLI Capability Benchmark Runner

The benchmark runner is a black-box runner over compiled `calctl` and `cald`
with fixed scoring contracts:

1. Load `tasks.jsonl` and `providers.json`.
2. Resolve available provider candidates.
3. Run CAL acquisition in replay or live mode.
4. In replay mode, run direct held-out reuse through `calctl runs create` when
   held-out fixture inputs satisfy the promoted binding contract.
5. Route held-out intents through Capability Use and execute the selected
   promoted bindings.
6. Score held-out outputs with `oracles/*.py`, not with CAL's generated
   verifier packages.
7. Report four evidence layers:
   - acquisition evidence;
   - held-out task success;
   - capability-layer evidence;
   - cost and reuse evidence.

Generated outputs belong under:

```text
evals/out/cli-capability/<run-id>/
```

## Replay Bootstrap

The first executable slice supports deterministic replay for `file_hash_sha1`:

```sh
python3 evals/cli-capability/runner/run.py \
  --mode replay \
  --tasks file_hash_sha1 \
  --calctl build/bin/calctl \
  --cald build/bin/cald
```

This slice is intentionally narrow. It proves the benchmark runner can collect
the four evidence layers for one clean multi-binding case. It records both
direct reuse and intent-level Use:

```text
promoted binding -> direct fixture reuse -> oracle
intent -> selected promoted binding -> oracle
```

The replay focus set runs the three current focus tasks:

```sh
python3 evals/cli-capability/runner/run.py \
  --mode replay \
  --level focus \
  --calctl build/bin/calctl \
  --cald build/bin/cald
```

## Live LLM Bootstrap

The same runner also supports a narrow live LLM focus mode for the same task:

```sh
CAL_LLM_API=chat_completions \
CAL_LLM_BASE_URL=<openai-compatible base url> \
CAL_LLM_MODEL=<model> \
CAL_LLM_API_KEY=<api key> \
python3 evals/cli-capability/runner/run.py \
  --mode live_llm \
  --level focus \
  --calctl build/bin/calctl \
  --cald build/bin/cald
```

Live mode does not use replay proposals. CAL asks the configured model to infer
candidate capabilities, verifier harnesses, and probe plans from observed CLI
behavior. The benchmark scores intent-level Use with fixed oracle scripts, not
with the generated verifier package.

Replay mode is strict and fails on any recorded benchmark failure. Live mode is
allowed to record failed providers or candidates, but the run is considered
usable only if at least one intent-level Use path passes the benchmark oracle.

For stability measurements, run live mode multiple times as separate benchmark
runs so each run gets its own `CAL_HOME` and run-id:

```sh
for i in 1 2 3; do
  python3 evals/cli-capability/runner/run.py \
    --mode live_llm \
    --level focus \
    --calctl build/bin/calctl \
    --cald build/bin/cald
done
```
