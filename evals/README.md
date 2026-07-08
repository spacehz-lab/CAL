# CAL Evals

`evals/` contains executable evaluation assets for CAL system claims.

This directory is separate from `tests/`:

- `tests/` catches engineering regressions.
- `evals/cli-capability/` defines the current executable evaluation surface for
  local CLI capability acquisition, capability-model evidence, held-out reuse,
  and baseline comparison.
- `evals/results/` stores curated, commit-ready result summaries when a run is
  intentionally selected for release notes or reports.

Generated outputs belong under `evals/out/`, which is ignored by git. Commit
eval runners, tasks, fixtures, scoring definitions, curated summaries, and
documentation.
Do not commit API keys, raw LLM responses, full provider output dumps, or local
machine-specific run artifacts.

Evaluation runs may produce ignored local outputs under `evals/out/`; reported
results should either point to the exact generated run directory used locally or
copy a compact summary into `evals/results/`.

## Layout

```text
evals/
  cli-capability/      # executable eval definition, fixtures, runner, scoring
  results/             # curated commit-ready result summaries
  out/                 # local generated outputs, ignored by git
```

`evals/cli-capability/suites/` physically separates the paper questions:

```text
acquisition.jsonl       # acquire verified bindings from real CLI surfaces
capability_model.jsonl  # prove Provider -> Capability* and Capability -> Binding*
reuse.jsonl             # held-out use/run plus independent oracle scoring
```

Benchmark reports should compare CAL with direct CLI oracle, LLM one-shot CLI,
and provider-tool baselines. The primary comparison is repeated-task
amortization: LLM calls, tokens, latency, and oracle-verified successes across
held-out reuse tasks.

## Current Eval

Replay CLI capability eval:

```sh
make build
python3 evals/cli-capability/runner/run.py \
  --mode replay \
  --level focus \
  --calctl build/bin/calctl \
  --cald build/bin/cald
```

Live LLM CLI capability eval:

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
