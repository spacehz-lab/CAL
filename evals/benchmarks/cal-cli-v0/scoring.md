# CAL CLI v0 Scoring

This scoring contract defines the target shape for the CAL CLI v0 seed
benchmark. The benchmark should not be marked active until the tasks, fixtures,
result schema, and runner all follow this contract.

## Case Outcomes

Each task case should end in exactly one top-level outcome:

- `success`: provider was available, at least one candidate was verified,
  promotion completed, Use selected a promoted binding for the held-out intent,
  runtime reuse completed on held-out inputs, and the benchmark oracle passed.
- `partial_success`: acquisition produced verified evidence and promotion, but
  a non-core reporting or optional verification step failed.
- `verification_failure`: a candidate was proposed and probed, but the
  deterministic verifier rejected the outcome.
- `promotion_failure`: verification passed, but Capability or Binding promotion
  failed.
- `reuse_failure`: promotion succeeded, but later `calctl runs create` failed.
- `use_failure`: promotion succeeded, but later `calctl use` could not select a
  promoted binding or could not produce binding-compatible inputs.
- `oracle_failure`: runtime reuse completed, but the benchmark oracle rejected
  the held-out output.
- `cli_unavailable`: the required provider could not be found or did not meet
  the declared environment requirement.
- `unsupported`: the task requires a provider behavior or verifier type outside
  the current v0 scope.
- `proposal_failure`: live LLM mode did not produce a usable candidate or probe
  plan.
- `runner_failure`: the benchmark infrastructure failed before the case reached
  CAL's acquisition loop.

Only `success` should count as oracle-verified reuse. `partial_success` may be
reported separately but must not inflate reuse success rate.

## Verification Boundary

CAL runtime verifier packages and benchmark oracles have different jobs:

- CAL verifier packages decide whether a candidate can be promoted into CAL's
  capability catalog.
- Capability Use decides whether a user or benchmark intent can route to a
  promoted binding and binding-compatible inputs.
- Benchmark oracles decide whether the promoted binding solved the held-out task
  on held-out reuse inputs.

This distinction is required because live LLM acquisition may generate verifier
packages. Reported results should never use an LLM-generated verifier as the only
success criterion for a benchmark task.

## Metrics

Aggregate benchmark output should report:

- attempted cases
- available providers
- unavailable providers
- candidate count
- probe pass count
- probe fail count
- promoted capabilities
- promoted bindings
- Use selections
- Use selection failures
- Use shortlist size
- verified reuses
- oracle-verified reuses
- capability id quality
- intent selection success rate
- failed cases by stage
- reuse success rate
- acquisition latency distribution
- Use latency distribution
- reuse latency distribution
- LLM call count and token count when available

Intent selection success rate is:

```text
successful_use_selections / held_out_intents
```

Oracle reuse success rate is:

```text
oracle_verified_reuses / successful_use_selections
```

Report provider availability separately so missing local CLIs do not look like
CAL acquisition failures.

## Baseline Scoring

Baseline results should use the same task ids and fixtures.

- Direct CLI oracle succeeds when the hand-authored invocation produces the
  expected deterministic verifier result.
- LLM one-shot CLI command succeeds when the generated command produces the
  expected deterministic verifier result without CAL promotion or reuse.
- CAL succeeds only when acquisition, deterministic verification, promotion,
  later Use selection, runtime reuse, and independent benchmark oracle scoring
  all succeed.

Reported benchmark summaries should compare baselines on task success, latency,
LLM calls/tokens, and whether successful behavior becomes reusable. Baselines
that do not promote bindings should have no verified reuse count.

## Failure Reporting

Every non-success case should record:

- task id
- provider id or provider path when available
- failure stage
- failure reason
- whether a candidate was produced
- whether a probe was executed
- verifier id and verifier result when available
- Use selection status, shortlist size, selected capability id, and selected
  binding id when available
- benchmark oracle id and oracle result when available
- whether any Capability or Binding record was promoted

Failure cases are part of the evidence. They show that promotion is gated by
deterministic verification rather than LLM assertion.

Capability id quality should be reported separately from task success. A live
LLM acquisition may generate a semantically usable capability id that differs
from the benchmark's preferred id. The benchmark should not treat naming drift
as a failed held-out reuse when Use can route the intent to the promoted binding
and the oracle passes.

Use resolver quality should also be reported separately from acquisition
quality. The v0 resolver uses a local high-recall topK shortlist plus one LLM
selection call. A failure should identify whether the correct binding was absent
from the shortlist, present but not selected, selected but locally invalid, or
selected and failed during Run. This keeps the current engineering choice
separate from the broader CAL claim and leaves room for future embedding-backed
retrieval or progressive detail fetch.
