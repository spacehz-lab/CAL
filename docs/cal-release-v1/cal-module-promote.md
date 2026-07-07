# CAL Release V1 Promote

`promote/` owns acquisition-time promotion from verified candidates into durable
capability records.

It is the boundary where a successful probe becomes a reusable
`Capability/Binding`. It does not perform verification itself.

## Goal

`promote/` turns passed probe results into durable reusable bindings:

```text
Candidates + Probes
-> select passed probes
-> build promoted Binding
-> create or update Capability
-> save Capability
-> model.Promotion
```

The normal acquisition flow is:

```text
entry -> observe -> proposal -> probe -> promote -> tracelog
```

`probe` decides whether a candidate passed. `promote` decides how a passed
candidate is represented as durable capability state.

## Boundary

`promote/` owns:

- Promotion request validation.
- Candidate and probe alignment by `Probe.CandidateIndex`.
- Filtering out non-passed probes.
- Rejecting passed probes that cannot become durable bindings.
- Stable binding id derivation from capability id, provider id, and execution.
- Construction of `model.Binding` with promoted state.
- Construction of new `model.Capability` records.
- Merging promoted bindings into existing capability records.
- Updating an existing binding with the same binding id.
- Saving the final capability through a narrow store interface.
- Returning `model.Promotion` records for trace writing.

`promote/` does not own:

- Provider registration or loading. That belongs to `entry`.
- Provider observation. That belongs to `observe`.
- Candidate generation or prompt behavior. That belongs to `proposal`.
- Probe workdir, fixture materialization, execution, or check classification.
  Those belong to `probe`.
- VerifySpec predicate semantics or check execution. That belongs to `check`.
- Provider execution. That belongs to `execute`.
- Trace persistence. That belongs to `tracelog`.
- Formal execution of promoted bindings. That belongs to `run`.
- Intent selection. That belongs to `use`.
- LLM calls.
- Store implementation details.
- Config loading, logging setup, API transport, HTTP handlers, CLI rendering, or
  daemon lifecycle.

Promotion should treat `model.Probe` as the trusted verification summary from
the previous stage. It must not re-run verification or reinterpret probe output.

## Dependency Rule

```text
promote -> model
```

`promote` defines a small local `Store` interface instead of importing
`store`. Runtime composition passes `*store.Store` because it satisfies the
interface.

Forbidden:

```text
promote -> store
promote -> proposal
promote -> probe
promote -> check
promote -> execute
promote -> acquisition
promote -> entry
promote -> observe
promote -> tracelog
promote -> run
promote -> use
promote -> eval
promote -> llm
promote -> config
promote -> logging
promote -> contract/httpserver/cli/cald
```

Callers:

```text
acquisition -> promote
cald/app    -> promote
```

`promote` consumes only durable `model` records: `Candidate`, `Probe`,
`Capability`, `Binding`, and `Promotion`.

## Files

First V1 should keep the package small:

```text
promote/
  runner.go
  request.go
  errors.go
```

`runner.go` owns the main promotion flow, target alignment, capability loading,
merge, save, and result assembly.

`request.go` owns `Request`, `Result`, `Target`, `Store`, and `Options` if
options become necessary.

`errors.go` owns promotion error codes, action constants, and coded errors.

Add `build.go` or `merge.go` only if `runner.go` becomes hard to read. Do not
split files just to mirror stage names.

## Public Shape

Use one dependency-owning runner:

```go
type Runner struct {
	store Store
	now   func() time.Time
}

func NewRunner(store Store, now func() time.Time) *Runner
func (r *Runner) Run(ctx context.Context, req *Request) (*Result, error)
```

`Runner` owns persistence, so callers should use `NewRunner`. Plain
request/result values may still be assembled with literals.

`request.go` owns:

```go
type Store interface {
	GetCapability(id string) (model.Capability, bool, error)
	SaveCapability(capability *model.Capability) error
}

type Request struct {
	Candidates []model.Candidate
	Probes     []model.Probe
}

type Result struct {
	Promotions []model.Promotion
}
```

`Target` is the internal per-passed-probe unit:

```go
type Target struct {
	CandidateIndex int
	Candidate      *model.Candidate
	Probe          *model.Probe
}
```

## Promotion Rules

Promote only passed probes:

```text
probe.Passed == true
```

Skip non-passed probes. A failed probe is already represented in trace data and
should not become a promotion error.

Reject a passed probe when:

- `CandidateIndex` is out of range.
- Candidate `ProviderID` is empty.
- Candidate `CapabilityID` is empty or invalid.
- Candidate `Description` is empty.
- Candidate `Execution.Kind` is empty.
- Probe `Verify.Level` is empty.
- Probe `Verify.Level` is `L0`.
- Probe `Verify.Method` is empty.
- Probe `Evidence` is empty.
- Binding id derivation fails.
- Final capability validation fails.

The binding id must be deterministic:

```go
model.BindingIDForExecution(candidate.CapabilityID, candidate.ProviderID, candidate.Execution)
```

Promoted bindings must use:

```go
State: model.BindingStatePromoted
```

Binding `Verify` comes from `probe.Verify`. Binding `Evidence` comes from
`probe.Evidence`.

`promote` must not call `check` to re-validate predicate semantics. That
semantic validation happened in `probe`. Promotion only enforces the durable
binding requirements needed to save a promoted capability.

## Merge Rules

When the capability does not exist:

```text
capability_action = created
binding_action    = created
```

When the capability exists:

```text
capability_action = reused
```

If the binding id already exists, replace that binding:

```text
binding_action = updated
```

If the binding id does not exist, append the binding:

```text
binding_action = created
```

Existing capability description should be preserved when non-empty. If the
existing description is empty, fill it from the candidate description.

## Action Constants

Use stable string constants:

```text
created
reused
updated
```

These values are written into `model.Promotion` and consumed by trace, API, and
eval code. They must not appear as ad hoc string literals in production logic.

## Error Codes

Use stable string constants in `errors.go`:

```text
invalid_promotion_input
promotion_rejected
promotion_store_failed
```

`invalid_promotion_input` is for bad request shape or missing runner
dependencies.

`promotion_rejected` is for a passed probe that cannot legally become a durable
binding.

`promotion_store_failed` is for capability load or save failures.

## Failure Behavior

Promotion is stricter than probe:

- Non-passed probes are skipped.
- A passed but invalid probe/candidate is an error.
- Store failures are errors.

This keeps the trace honest: a candidate can fail probing without breaking the
promotion stage, but a claimed passed probe must either become a valid durable
binding or produce a promotion error.

## Completion Criteria

Promote implementation is complete when:

- It promotes only passed probes.
- It rejects L0, missing evidence, invalid capability ids, missing
  descriptions, and missing execution kind.
- It derives stable binding ids from execution identity.
- It creates new capabilities.
- It appends different bindings to existing capabilities.
- It updates existing bindings with the same id.
- It preserves existing non-empty capability descriptions.
- It returns correct `model.Promotion` action values.
- It imports only `internal/model` plus the Go standard library.
- Unit tests cover create, append, update, skip failed probe, reject invalid
  passed probe, store load failure, and save failure.
