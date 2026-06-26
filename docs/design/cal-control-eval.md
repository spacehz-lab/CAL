# CAL Eval Control

Eval Control is the evaluation access module for CAL experiments.

It summarizes whether acquired capabilities were discovered, promoted, reused, and verified in a way that improves later agent tasks.

## Role

Eval answers one question:

```text
Did CAL acquisition and reuse improve task execution compared with task-centric execution?
```

It is not part of the normal online agent path.

The normal online path remains:

```text
calctl use
-> calctl runs create
```

Eval reads records produced by Discovery, Use, and Run. It should not mutate
Trace, Capability, Binding, Use, or Run records.

## Input

Default command:

```bash
calctl eval --json
```

The current command computes aggregate metrics over all local records.

Future filters may be added when the record set is large:

```bash
calctl eval --capability-id <capability-id> --json
calctl eval --provider-id <provider-id> --json
```

Those filters are not part of the current command surface.

## Source Records

Eval reads:

```text
Trace
Use
Run
Capability
Binding
Provider
```

Trace contributes acquisition metrics:

```text
targeted acquisition attempt count
completed/failed targeted attempt count
candidates count
probe count
probe pass/fail/ambiguous counts
promoted binding count
per-capability acquisition counts
per-source acquisition counts
failure reasons
LLM proposer call count when recorded
discovery duration
```

For multi-capability acquisition, one Trace can contain multiple candidates,
probes, and promotions. Eval counts promotions from `promotions[]`.

Provider Entry traces contribute to `summary.traces`, but they are not counted
as targeted acquisition attempts unless they include a hint, candidate, probe,
promotion, or acquisition error.

Run contributes reuse metrics:

```text
run count
run success/failure count
selected capability id
selected binding id
selected provider id
verified run count
verified success/failure over verified runs
duration
error code
evidence count
```

Use contributes intent-routing metrics when Use records are available:

```text
use count
use success/failure count
selected capability id
selected binding id
selected provider id
assigned control input keys
resolver source
duration
error code
```

Capability and Binding records provide current promoted surface size and verifier coverage.

## Output

Output shape:

```json
{
  "summary": {
    "capabilities": 6,
    "promoted_bindings": 9,
    "traces": 12,
    "runs": 42
  },
  "acquisition": {
    "attempt_count": 18,
    "completed_count": 13,
    "failed_count": 5,
    "promotion_count": 9,
    "capability_created_count": 4,
    "capability_reused_count": 5,
    "binding_created_count": 7,
    "binding_updated_count": 2,
    "candidate_count": 18,
    "probe_count": 14,
    "probe_pass_count": 9,
    "probe_fail_count": 5,
    "binding_promotion_rate": 0.5,
    "probe_success_rate": 0.64,
    "by_capability": [
      {
        "capability_id": "document.export_pdf",
        "attempts": 10,
        "completed": 7,
        "failed": 3,
        "promotions": 7,
        "candidates": 10,
        "probes": 10,
        "probe_passes": 7,
        "probe_failures": 3
      }
    ],
    "by_source": [
      {
        "source": "proposal:replay",
        "attempts": 4,
        "completed": 3,
        "failed": 1,
        "promotions": 3,
        "candidates": 4,
        "probes": 4,
        "probe_passes": 3,
        "probe_failures": 1
      }
    ]
  },
  "reuse": {
    "run_count": 42,
    "run_success_count": 39,
    "run_failure_count": 3,
    "verified_run_count": 14,
    "verifier_fail_count": 1,
    "run_success_rate": 0.93,
    "verified_success_rate": 0.86,
    "verifier_failure_rate": 0.07,
    "avg_run_duration_ms": 930
  },
  "use": {
    "use_count": 12,
    "use_success_count": 10,
    "use_failure_count": 2,
    "intent_selection_success_rate": 0.83,
    "avg_use_duration_ms": 240
  }
}
```

Fields may be omitted when the source records do not contain enough data. Missing data must not be silently treated as zero.

## Metrics

Current first-version metrics are the aggregate, acquisition, use, and reuse
fields shown above. Broader experiment metrics may include:

```text
acquisition_precision
acquisition_cost
safe_probe_success_rate
outcome_verifier_pass_rate
binding_promotion_rate
capability_reuse_rate
intent_selection_success_rate
run_success_rate
verified_success_rate_after_reuse
average_use_duration_ms
average_run_duration_ms
llm_proposer_call_count
estimated_llm_proposer_calls_saved
estimated_low_level_actions_saved
drift_detection_rate
```

Not every metric is required in the first implementation. The first version should compute only metrics directly supported by stored records.

## Baseline

Eval may compare against baseline records when available:

```text
task-centric GUI or shell execution without CAL reuse
manual capability bindings
provider-specific action replay
```

Baseline data is optional in the control plane. If no baseline exists, Eval should report CAL-only acquisition and reuse metrics rather than inventing improvement estimates.

## Boundary

Eval owns:

```text
reading Trace and Run records
reading Use records when available
aggregating acquisition and reuse metrics
reporting missing-data gaps
emitting stable JSON for experiments
```

Eval does not own:

```text
Discovery Inference
Discovery Verification
Discovery Promotion
capabilities listing
capability execution
baseline generation
LLM proposer judging of task success
record mutation
```

## Evaluation Use

Eval is the bridge from the engineering system to reported evaluation claims.

It should support the discover-once, reuse-later protocol:

```text
Phase A: no acquired capability
Phase B: discover and promote bindings
Phase C: route future intents through Use and run promoted bindings
Phase D: measure acquisition, reuse, cost, and success
```

Eval should make it possible to report whether CAL reduced future low-level action count, `LLM proposer` calls, latency, and verification cost while preserving outcome verification.
