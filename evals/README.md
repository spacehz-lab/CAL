# CAL Evals

`evals/` contains executable evaluation assets for CAL system claims.

This directory is separate from `tests/`:

- `tests/` catches engineering regressions.
- `evals/experiments/` records exploratory evidence for local release checks.
- `evals/benchmarks/` defines stable task sets, scoring rules, and baselines for
  fixed seed benchmarks and future comparative evaluation.
- `evals/artifacts/` stores sanitized release summaries and selected trace
  excerpts for reported results.

Generated outputs belong under `evals/out/`, which is ignored by git. Commit
experiment runners, cases, fixtures, scoring definitions, and documentation.
Do not commit API keys, raw LLM responses, full provider output dumps, or local
machine-specific run artifacts.

Evaluation runs may produce ignored local outputs under `evals/out/`; reported
results should point to a sanitized committed summary or trace excerpt for the
exact run used.

## Layout

```text
evals/
  experiments/
    cli-matrix/

  benchmarks/
    cal-cli-v0/

  artifacts/           # sanitized release artifacts
  out/                 # local generated outputs, ignored by git
```

## Current Experiments

Replay real-CLI matrix:

```sh
make build
python3 evals/experiments/cli-matrix/run.py \
  --mode replay \
  --level full \
  --calctl build/bin/calctl \
  --cald build/bin/cald
```

Live LLM real-CLI matrix:

```sh
CAL_LLM_API=chat_completions \
CAL_LLM_BASE_URL=<openai-compatible base url> \
CAL_LLM_MODEL=<model> \
CAL_LLM_API_KEY=<api key> \
  python3 evals/experiments/cli-matrix/run.py \
    --mode live_llm \
    --level smoke \
    --calctl build/bin/calctl \
    --cald build/bin/cald
```
