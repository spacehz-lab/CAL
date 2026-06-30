# CLI Capability Benchmark

This directory defines the first stable CAL seed benchmark surface for release
evaluation.

Benchmark id: `cli-capability`
Benchmark version: `v0`

This eval fixes tasks, providers, scoring, and baselines so results can be
compared across runs and systems.

The v0 benchmark should stay small. Its job is to make CAL's first
trace-backed acquisition/reuse evidence reproducible, not to become a broad
computer-use leaderboard.

## Planned Contents

```text
evals/cli-capability/
  README.md
  tasks.jsonl      # fixed task intents and held-out reuse fixtures
  providers.json   # provider availability and environment requirements
  fixtures/        # deterministic task inputs
  oracles/         # independent benchmark scoring scripts
  scoring.md       # scoring contract and failure taxonomy
  baselines/       # baseline runners or baseline result format
  runner/          # benchmark runner
  report/          # report generation notes and templates
```

## Scoring Boundary

The benchmark separates CAL's internal promotion gate from external scoring:

```text
CAL verify spec  -> decides whether a candidate may be promoted
benchmark oracle -> decides whether a held-out benchmark task succeeded
```

Live LLM mode may propose verify specs, probe plans, and capability ids. The
benchmark must not predeclare acceptable capability ids and must not trust CAL's
internal verify checks as final task evidence. Held-out reuse outputs are scored
by fixed scripts under `oracles/`.

## Seed Benchmark Scope

The seed benchmark should cover:

- 10 fixed real-CLI tasks across production CLIs;
- a replay mode that runs without LLM API keys;
- a live LLM focus set of 3-5 tasks;
- at least one provider that promotes more than one capability binding;
- at least one capability that can be realized by two provider bindings;
- at least one failed or rejected candidate record;
- deterministic verify-check evidence for every promoted binding;
- intent-level held-out reuse through `calctl use`, with `calctl runs create`
  remaining the deterministic lower-level primitive.

The current checked-in task schema is a bootstrap slice with five task intents.
It is meant to lock the benchmark contract and the four evidence layers before
expanding to the full 10-task release run.

This benchmark should report four evidence layers:

- acquisition evidence: observation, candidate generation, probe execution,
  deterministic verification, and promotion;
- held-out task success: replay direct binding reuse for deterministic
  engineering checks, live intent-level Use selection, runtime execution on
  reuse fixtures, and benchmark oracle scoring;
- capability-layer evidence: provider-to-capability and capability-to-binding
  structure;
- cost and reuse evidence: acquisition latency, reuse latency, LLM calls, token
  count when available, and run-stage LLM calls.

For v0, Use selection is scored as a bounded resolver:

```text
local topK shortlist over promoted bindings
-> one LLM selection call
-> local validation
-> Run
```

The benchmark should record shortlist size and selection status, but the
benchmark claim should not depend on this being the final retrieval design.
Future runs may replace the shortlist stage with embedding-backed retrieval or
progressive detail fetch while keeping the same held-out oracle scoring.

## Required Result Fields

Benchmark results should be machine-readable and include:

- run id, mode, platform, architecture, CAL revision, and model settings when
  live LLM mode is used;
- selected tasks, attempted providers, available providers, and unavailable
  provider reasons;
- candidate count, probe pass count, probe fail count, promoted capabilities,
  promoted bindings, Use selections, verified reuses, and failed cases;
- per-case provider path, observation sources, candidate capability id, binding
  execution kind, verification level, probe status, promotion action, Use selection
  status, Use shortlist size, selected capability id, selected binding id,
  reuse status, benchmark oracle status, failure stage, and failure reason;
- acquisition latency, Use latency, reuse latency, LLM call count, and token
  count when available.

## Baselines

The minimum v0 baselines are:

- direct CLI oracle: a hand-authored correct invocation for each task, used as a
  correctness and latency upper-bound reference;
- LLM one-shot CLI command: the model receives provider documentation and task
  input, then emits a command without CAL promotion or reuse;
- CAL replay/live: the CAL acquisition loop with deterministic verification,
  promotion, later intent-level Use, and replay-only direct runtime reuse.

The oracle baseline is not a fair agent baseline. It exists to show task
feasibility and provide a performance reference.

No benchmark result is committed by default. Generated results should be written
under `evals/out/cli-capability/`.

Each run writes:

- `flow.json`: the primary step-by-step evidence artifact, organized around
  provider resolution, registration, the four acquisition stages, direct reuse,
  and intent-level Use;
- `summary.json`: aggregate task/provider metrics derived from the run records;
- `index.html`: a human-readable flow report with a closed-loop matrix and
  acquisition-stage detail;
- `cald.log`: local service log for debugging.

For reported release results, reference the exact generated run directory and
keep API keys, raw secret-bearing prompts, and machine-specific dumps out of git.
