# CLI Capability Benchmark

This directory defines the first stable CAL seed benchmark surface for release
evaluation.

Benchmark id: `cli-capability`
Benchmark version: `v0`

This eval fixes suites, cases, providers, scoring, and baselines so results can
be compared across runs and systems.

The v0 benchmark should stay small. Its job is to make CAL's first
trace-backed acquisition/reuse evidence reproducible, not to become a broad
computer-use leaderboard.

## Planned Contents

```text
evals/cli-capability/
  README.md
  suites/          # acquisition, capability-model, and reuse suite cases
  providers.json   # provider availability and environment requirements
  fixtures/        # deterministic case inputs
  oracles/         # independent benchmark scoring scripts
  scoring.md       # scoring contract and failure taxonomy
  baselines/       # baseline runners or baseline result format
  runner/          # benchmark runner
  report/          # report generation notes and templates
```

## Suite Model

The benchmark is physically split into three suites:

```text
suites/acquisition.jsonl
suites/capability_model.jsonl
suites/reuse.jsonl
```

The suites map directly to the arXiv v0 experiment questions:

- Acquisition: can CAL acquire verified provider-specific bindings from real CLI
  surfaces? This suite does not compare against non-CAL baselines.
- Capability Model: does CAL form a capability layer instead of a one-provider
  wrapper? This suite shows structural difference: CAL has durable
  Provider -> Capability -> Binding records while non-CAL baselines do not.
- Reuse: can promoted bindings solve held-out cases through `calctl use` and
  `calctl runs create` without another acquisition step? This suite is the main
  comparison against non-CAL baselines.

Each suite case should reference shared provider records, fixtures, replay
proposals, oracles, and baseline definitions. `suites/*.jsonl` is the only
benchmark input.

## Scoring Boundary

The benchmark separates CAL's internal promotion gate from external scoring:

```text
CAL verify spec  -> decides whether a candidate may be promoted
benchmark oracle -> decides whether a held-out benchmark case succeeded
```

Live LLM mode may propose verify specs, probe plans, and capability ids. The
benchmark must not predeclare acceptable capability ids and must not trust CAL's
internal verify checks as final case evidence. Held-out reuse outputs are scored
by fixed scripts under `oracles/`.

## Seed Benchmark Scope

The seed benchmark should cover:

- 15-20 fixed real-CLI cases across production CLIs for the full release report;
- 8-10 focus cases for repeated live LLM runs;
- a replay mode that runs without LLM API keys;
- a live LLM focus set of 3-5 cases;
- at least one provider that promotes more than one capability binding;
- at least one capability that can be realized by two provider bindings;
- at least one failed or rejected candidate record;
- deterministic verify-check evidence for every promoted binding;
- intent-level held-out reuse through `calctl use`, with `calctl runs create`
  remaining the deterministic lower-level primitive.

The current checked-in suite files are a bootstrap slice. They lock the
benchmark contract and evidence layers before expanding into the full release
run.

Representative provider coverage:

| Domain | Providers | Target capabilities |
| --- | --- | --- |
| Text and encoding | `base64`, `openssl` | Base64 encode/decode |
| Security and checksums | `shasum`, `openssl` | SHA-1 checksum |
| Archive and compression | `tar`, `zip`, `gzip`, `ditto` | archive create, compression |
| Structured data | `jq`, `python3`, `plutil`, `yq` | JSON query, plist conversion, YAML query |
| Documents | `pandoc`, `pdftotext` | document conversion, PDF text extraction |
| Media and images | `ffmpeg`, `magick` | media conversion, image resize/convert |
| Search and extraction | `rg` | text search |

Optional third-party CLIs may be unavailable on a host. The runner should report
those cases as provider availability, not CAL acquisition failure.

This benchmark should report four evidence layers:

- acquisition evidence: observation, candidate generation, probe execution,
  deterministic verification, and promotion;
- held-out case success: replay direct binding reuse for deterministic
  engineering checks, live intent-level Use selection, runtime execution on
  reuse fixtures, and benchmark oracle scoring;
- capability-layer evidence: provider-to-capability and capability-to-binding
  structure;
- cost and reuse evidence: acquisition latency, reuse latency, LLM calls, token
  count when available, and run-stage LLM calls.

The HTML report should render those layers as three suite sections:

- Acquisition Suite: acquisition matrix, proposal stage detail, probe/promote
  evidence, and acquisition failure taxonomy. It should not display baseline
  comparison as a pass/fail claim.
- Capability Model Suite: provider coverage, capability coverage, and
  provider-to-capability-to-binding tables. Baselines appear only as a
  no-durable-structure reference, not as a correctness or cost comparison.
- Reuse Suite: held-out reuse matrix, Use resolver detail, CAL verify vs
  independent oracle, and baseline cost amortization. This is the only suite
  where non-CAL methods are scored as primary comparisons.

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
- selected cases, attempted providers, available providers, and unavailable
  provider reasons;
- candidate count, probe pass count, probe fail count, promoted capabilities,
  promoted bindings, Use selections, verified reuses, and failed cases;
- per-case provider path, observation sources, candidate capability id, binding
  execution kind, verification level, probe status, promotion action, Use selection
  status, Use shortlist size, selected capability id, selected binding id,
  reuse status, benchmark oracle status, failure stage, and failure reason;
- acquisition latency, Use latency, reuse latency, LLM call count, and token
  count when available.

`summary.json` should aggregate by suite:

```json
{
  "suites": {
    "acquisition": {
      "attempted": 0,
      "available": 0,
      "candidates": 0,
      "probe_passed": 0,
      "probe_failed": 0,
      "promoted_bindings": 0
    },
    "capability_model": {
      "providers": 0,
      "capabilities": 0,
      "multi_capability_providers": 0,
      "multi_binding_capabilities": 0
    },
    "reuse": {
      "held_out_uses": 0,
      "use_selected": 0,
      "runs_succeeded": 0,
      "oracle_passed": 0
    }
  }
}
```

## Baselines

Baselines are a horizontal method dimension, not a fourth suite. The current
checked-in runner implements:

- direct CLI oracle: a hand-authored correct invocation for each case, used as a
  correctness and latency upper-bound reference.

The planned v0 baseline set also includes:

- LLM one-shot CLI command: the model receives provider documentation and case
  input, then emits a command without CAL promotion or reuse.
- provider tool baseline: the model treats the provider CLI as a generic tool and
  selects arguments for every case without durable promotion.
- CAL replay/live: the CAL acquisition loop with deterministic verification,
  promotion, later intent-level Use, and replay-only direct runtime reuse.

The oracle baseline is not a fair agent baseline. It exists to show case
feasibility and provide a performance reference.

Baselines are used as follows:

- Acquisition Suite: no baseline comparison; report CAL acquisition quality.
- Capability Model Suite: show that non-CAL methods have no durable capability
  structure; do not use baseline pass/fail as the main claim.
- Reuse Suite: compare CAL with direct CLI now, and with LLM one-shot and
  provider-tool methods once those runners are implemented.

The main comparison is repeated-case amortization in the Reuse Suite:

```text
method / repeated cases / LLM calls / tokens / total latency / oracle successes
-> average cost per oracle-verified success
```

CAL may lose on first-case latency because acquisition has an upfront cost. The
claim is that promoted bindings reduce repeated command-synthesis work on later
held-out cases.

No benchmark result is committed by default. Generated results should be written
under `evals/out/cli-capability/`.

Replay focus run:

```sh
python3 evals/cli-capability/runner/run.py \
  --mode replay \
  --suite acquisition,capability_model,reuse \
  --level focus \
  --calctl build/bin/calctl \
  --cald build/bin/cald
```

Single suite run:

```sh
python3 evals/cli-capability/runner/run.py \
  --mode replay \
  --suite acquisition \
  --case file_hash_sha1 \
  --level focus
```

Each run writes:

- `summary.json`: aggregate metrics and suite-level scores derived from the run
  records;
- `flow.json`: the primary step-by-step evidence artifact, organized around
  provider resolution, registration, acquisition stages, direct reuse,
  intent-level Use, oracle scoring, and reuse-suite baseline comparison;
- `index.html`: a human-readable flow report with a closed-loop matrix and
  acquisition evidence, capability-model structure, reuse comparison, and
  cost-amortization sections;
- `artifact.json`: optional sanitized trace/run/capability excerpts used by a
  release report;
- `cald.log`: local service log for debugging.

For reported release results, reference the exact generated run directory and
keep API keys, raw secret-bearing prompts, and machine-specific dumps out of git.
