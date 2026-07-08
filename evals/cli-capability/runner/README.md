# CLI Capability Benchmark Runner

The runner is a suite-first black-box benchmark over compiled `calctl` and
`cald`. It loads only physical suite files. Each benchmark case lives in one
physical suite file under `suites/`.

## Flow

For each selected case and provider, the runner records:

```text
provider.resolve
-> provider.register
-> acquisition.run
-> acquisition.observe
-> proposal.surface
-> proposal.capability
-> proposal.binding
-> proposal.evidence
-> acquisition.probe
-> acquisition.promote
-> direct_reuse.oracle
-> intent_use.oracle
```

The runner scores held-out outputs with `oracles/*.py`. CAL internal verify
results are acquisition evidence, not benchmark truth.

Generated outputs belong under:

```text
evals/out/cli-capability/<run-id>/
```

Primary artifacts:

- `flow.json`: per-case flow, provider steps, candidates, reuse, and baselines.
- `summary.json`: aggregate metrics for scripts and tables.
- `artifact.json`: compact release artifact with trace references.
- `index.html`: human-readable report.

## Replay

Replay uses fixed proposals from `proposals/replay/<case-id>/<provider>.json`:

```sh
python3 evals/cli-capability/runner/run.py \
  --mode replay \
  --suite acquisition \
  --case file_hash_sha1 \
  --calctl build/bin/calctl \
  --cald build/bin/cald
```

Run the focus set:

```sh
python3 evals/cli-capability/runner/run.py \
  --mode replay \
  --level focus \
  --calctl build/bin/calctl \
  --cald build/bin/cald
```

Replay mode is strict: any benchmark failure fails the run.

## Live LLM

Live mode asks the configured LLM to produce proposal evidence from observed CLI
behavior:

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

Live mode records failed providers and candidates as evidence. The run is useful
only when at least one intent-level held-out use passes the benchmark oracle.
