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
eval runners, suite cases, fixtures, scoring definitions, curated summaries, and
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

`evals/cli-capability/scenarios/` defines the paper questions:

```text
acquisition.jsonl           # acquire verified bindings from real CLI surfaces
failure_gating.jsonl        # block invalid candidate bindings
repeated_reuse.jsonl        # held-out use plus independent oracle scoring
capability_structure.jsonl  # prove Provider -> Capability* and Capability -> Binding*
```

Benchmark reports should include a broad reuse-effectiveness profile and a
focused CAL reuse vs LLM one-shot comparison profile. The primary comparison is
repeated-case amortization: LLM calls, tokens, latency, and oracle-verified
successes across held-out reuse cases.

## Current Eval

Replay CLI capability eval:

```sh
make build
python3 evals/cli-capability/runner/run.py \
  --mode replay \
  --experiment acquisition,verification_failure,repeated_reuse,capability_structure \
  --level focus \
  --jobs 8 \
  --calctl build/bin/calctl \
  --cald build/bin/cald
```

Live LLM acquisition eval:

```sh
CAL_LLM_API=chat_completions \
CAL_LLM_BASE_URL=<openai-compatible base url> \
CAL_LLM_MODEL=<model> \
CAL_LLM_API_KEY=<api key> \
  python3 evals/cli-capability/runner/run.py \
    --mode live_llm \
    --experiment acquisition \
    --level full \
    --jobs 8 \
    --llm-jobs 8 \
    --calctl build/bin/calctl \
    --cald build/bin/cald
```

Live LLM capability-structure eval should include acquisition in the selected
experiments so it performs real acquisition instead of seed-only structure
calculation:

```sh
CAL_LLM_API=chat_completions \
CAL_LLM_BASE_URL=<openai-compatible base url> \
CAL_LLM_MODEL=<model> \
CAL_LLM_API_KEY=<api key> \
  python3 evals/cli-capability/runner/run.py \
    --mode live_llm \
    --experiment acquisition,capability_structure \
    --tag capability_structure \
    --level full \
    --jobs 8 \
    --llm-jobs 8 \
    --calctl build/bin/calctl \
    --cald build/bin/cald
```

Curated public result bundles are indexed in
`evals/results/cli-capability/README.md`. Those committed bundles are sanitized
summaries only; raw local runs remain under ignored `evals/out/cli-capability/`.
