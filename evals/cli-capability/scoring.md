# CLI Capability Scoring

This scoring contract defines the target shape for the CLI Capability seed
benchmark. The benchmark should not be marked active until the cases, fixtures,
result schema, and runner all follow this contract.

## Case Outcomes

Each suite case should end in exactly one top-level outcome:

- `success`: provider was available, at least one candidate was verified,
  promotion completed, Use selected a promoted binding for the held-out intent,
  runtime execution completed on held-out inputs, and the benchmark oracle
  passed.
- `partial_success`: acquisition produced verified evidence and promotion, but
  a non-core reporting or optional verification step failed.
- `verification_failure`: a candidate was proposed and probed, but the
  deterministic verify checks rejected the outcome.
- `promotion_failure`: verification passed, but Capability or Binding promotion
  failed.
- `reuse_failure`: in replay mode, promotion succeeded, fixture inputs
  satisfied the selected binding contract, but later `calctl runs create`
  failed.
- `use_failure`: promotion succeeded, but later `calctl use` could not select a
  promoted binding or could not produce binding-compatible inputs.
- `oracle_failure`: runtime execution completed, but the benchmark oracle rejected
  the held-out output.
- `cli_unavailable`: the required provider could not be found or did not meet
  the declared environment requirement.
- `unsupported`: the case requires a provider behavior or verification type outside
  the current v0 scope.
- `skipped`: in replay diagnostics, promotion succeeded, but a direct reuse
  fixture did not contain the runtime inputs required by that binding. Skipped
  direct reuses are reported but are not counted as oracle failures.
- `proposal_failure`: live LLM mode did not produce a usable candidate or probe
  plan.
- `runner_failure`: the benchmark infrastructure failed before the case reached
  CAL's acquisition loop.

Only `success` should count as oracle-verified reuse. `partial_success` may be
reported separately but must not inflate reuse success rate.

## Verification Boundary

CAL runtime verify specs and benchmark oracles have different jobs:

- CAL verify specs decide whether a candidate can be promoted into CAL's
  capability catalog.
- Capability Use decides whether a user or benchmark intent can route to a
  promoted binding and binding-compatible inputs.
- Benchmark oracles decide whether the promoted binding solved the held-out case
  on held-out reuse inputs.

This distinction is required because live LLM acquisition may propose internal
verify specs. Reported results should never use CAL's internal verify checks as
the only success criterion for a benchmark case.

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
- promoted bindings with replay direct reuse
- oracle-verified replay direct reuses
- oracle-verified intent uses
- capability id quality
- intent selection success rate
- failed cases by stage
- reuse success rate
- provider acquisition total and average latency
- proposal LLM total and average latency
- acquisition local overhead latency
- intent Use total and average latency
- replay direct reuse latency
- LLM call count and token count when available
- suite-level acquisition, capability-model, and reuse summaries
- reuse-suite baseline success, latency, LLM-call, token, and amortized cost
  comparison

The runner also emits a derived `scores` object for display and run-to-run
comparison. Scores are computed only from `summary` and must not affect
`validate_result()` pass/fail behavior.

Score timing fields use workflow names rather than a single opaque latency:

- `provider_acquisition_*`: total `acquisition run` time per provider;
- `proposal_llm_*`: model time spent producing the acquisition proposal;
- `acquisition_local_overhead_*`: provider acquisition time outside proposal
  LLM time;
- `intent_use_*`: held-out `calctl use` time;
- `replay_direct_reuse_*`: replay-only direct `calctl runs create` time.

Intent selection success rate is:

```text
successful_use_selections / held_out_intents
```

Oracle direct reuse success rate applies to replay mode and excludes skipped
direct reuses:

```text
oracle_verified_direct_reuses / held_out_direct_reuses
```

Oracle intent use success rate is:

```text
oracle_verified_intent_uses / held_out_intents
```

Report provider availability separately so missing local CLIs do not look like
CAL acquisition failures.

## Suite Scoring

The benchmark reports three suite groups:

### Acquisition

Acquisition cases score the path from provider surface to promoted binding:

```text
provider.resolve
-> provider.register
-> acquisition.observe
-> proposal.surface
-> proposal.capability
-> proposal.binding
-> proposal.evidence
-> probe
-> promote
```

Required metrics:

- attempted and available providers;
- candidate count;
- probe pass and fail count;
- promoted binding count;
- verification level distribution;
- acquisition latency and proposal LLM latency;
- failure stage and reason.

Acquisition Suite does not score non-CAL baselines. A direct CLI command can
prove the case is feasible elsewhere, but it is not an acquisition comparison.

### Capability Model

Capability-model cases score whether CAL builds structure above providers:

```text
Provider -> Capability*
Capability -> Binding* -> Provider
```

Required metrics:

- provider count;
- capability count;
- bindings per capability;
- capabilities per provider;
- multi-capability provider count;
- multi-binding capability count;
- verification level distribution by capability.

Capability Model Suite reports structural evidence. Non-CAL one-shot or
provider-tool methods have no durable Capability or Binding records, so their
role is a structural absence reference, not a pass/fail or latency comparison.

### Reuse

Reuse cases score held-out capability use:

```text
held-out intent
-> calctl use selection
-> runtime run
-> CAL verify
-> independent oracle
```

Required metrics:

- held-out use attempts;
- selection success count;
- shortlist size distribution;
- selected capability and binding ids;
- run success count;
- CAL verification pass count;
- independent oracle pass count;
- use and run latency;
- failure stage and reason.

## Baseline Scoring

Baseline results should use the same case ids and fixtures as Reuse Suite cases.

- Direct CLI oracle succeeds when the hand-authored invocation produces the
  expected deterministic oracle result.
- Planned LLM one-shot CLI command succeeds when the generated command produces the
  expected deterministic oracle result without CAL promotion or reuse.
- Planned provider tool baseline succeeds when the model can select a provider command
  and arguments for the current case and the independent oracle passes, without
  creating a durable binding.
- CAL succeeds only when acquisition, deterministic verification, promotion,
  later Use selection, runtime execution, and independent benchmark oracle
  scoring all succeed. Replay mode additionally requires direct binding reuse.

Reported benchmark summaries should compare baselines on held-out reuse success,
latency, LLM calls/tokens, and whether successful behavior becomes reusable.
Baselines that do not promote bindings should have no verified reuse count.
Suite manifests may declare only implemented baseline runners; unimplemented
planned baselines should remain documented design targets, not runnable case
configuration.

Repeated-case cost should be reported separately from first-case latency. CAL is
allowed to pay acquisition cost once; the paper comparison should show average
cost per oracle-verified success after multiple held-out reuse cases.

The paper-level framing is:

```text
Acquisition Suite: CAL can learn verified CLI bindings.
Capability Model Suite: learned bindings form a durable capability structure.
Reuse Suite: the durable structure beats repeated one-shot command synthesis on
held-out reuse cost, stability, and reusable records.
```

## Failure Reporting

Every non-success case should record:

- case id
- provider id or provider path when available
- failure stage
- failure reason
- whether a candidate was produced
- whether a probe was executed
- verification level, check summary, and verification result when available
- Use selection status, shortlist size, selected capability id, and selected
  binding id when available
- benchmark oracle id and oracle result when available
- whether any Capability or Binding record was promoted

Failure cases are part of the evidence. They show that promotion is gated by
deterministic verification rather than LLM assertion.

In live LLM mode, provider-level acquisition failures and rejected candidates
are reported as negative evidence, not closed-loop case failures, when another
promoted binding still lets `calctl use` solve the held-out intent and pass the
benchmark oracle.

Capability id quality should be reported separately from case success. A live
LLM acquisition may generate a semantically usable capability id that differs
from another run's naming. The benchmark should not predeclare acceptable
capability ids and should not treat naming drift as a failed held-out reuse when
Use can route the intent to the promoted binding and the oracle passes.

Use resolver quality should also be reported separately from acquisition
quality. The v0 resolver uses a local high-recall topK shortlist plus one LLM
selection call. A failure should identify whether the correct binding was absent
from the shortlist, present but not selected, selected but locally invalid, or
selected and failed during Run. This keeps the current engineering choice
separate from the broader CAL claim and leaves room for future embedding-backed
retrieval or progressive detail fetch.
