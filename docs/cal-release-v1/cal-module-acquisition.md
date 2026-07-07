# CAL Release V1 Acquisition

`acquisition/` owns the main targeted acquisition runner.

It is an orchestration package. It does not own provider registration,
observation details, proposal prompts, probe execution, promotion rules, trace
assembly, or storage paths.

## Goal

`acquisition/` turns one targeted provider acquisition request into a completed
or failed trace:

```text
provider id
-> load provider
-> start trace
-> load capability catalog
-> observe provider
-> propose candidates and probe plans
-> probe candidates
-> promote passed probes
-> complete trace
```

The normal flow is:

```text
entry -> observe -> proposal -> probe -> promote -> tracelog
```

`acquisition` owns the order, partial state, and failure trace behavior across
those packages.

## Boundary

`acquisition/` owns:

- Targeted acquisition request validation.
- Provider loading through `entry`.
- Capability catalog loading through a narrow local interface.
- Calling each acquisition stage in order.
- Passing trace id, provider, catalog, observations, candidates, probe plans,
  probes, promotions, and work root between packages.
- Preserving partial stage outputs when a later stage fails.
- Calling `tracelog.Start`, `tracelog.Complete`, `tracelog.Fail`, or
  `tracelog.Cancel`.
- Returning the final trace result to callers.

`acquisition/` does not own:

- Provider path resolution, registration, or durable provider writes. Those
  belong to `entry`.
- Provider-kind observation or CLI usage parsing. Those belong to `observe` and
  `observe/cli`.
- Proposal prompt wording, LLM calls, replay fixtures, deterministic proposal
  rules, proposal concurrency, or candidate selection. Those belong to
  `proposal`.
- Probe workdir layout below the provided work root, fixture materialization,
  execution, deterministic checks, timeouts, or probe classification. Those
  belong to `probe`.
- Promotion eligibility, capability merging, binding id derivation, or durable
  capability writes. Those belong to `promote`.
- Trace record assembly or trace persistence. Those belong to `tracelog`.
- Store path construction, JSON files, or atomic writes. Those belong to
  `store`.
- API, HTTP, local client, CLI, daemon lifecycle, config loading, logging setup,
  or LLM client construction.
- A generic workflow framework, `Stage` abstraction, or reusable `Flow`
  abstraction.

## Dependency Rule

```text
acquisition -> model
acquisition -> entry
acquisition -> observe
acquisition -> proposal
acquisition -> probe
acquisition -> promote
acquisition -> tracelog
```

`acquisition` should not import `store`. It defines narrow local interfaces for
the store-facing capabilities it needs, and runtime composition passes
`*store.Store` because it satisfies those interfaces.

Forbidden:

```text
acquisition -> store
acquisition -> check
acquisition -> execute
acquisition -> observe/cli
acquisition -> proposal/surface
acquisition -> proposal/capability
acquisition -> proposal/binding
acquisition -> proposal/evidence
acquisition -> proposal/policy
acquisition -> llm
acquisition -> config
acquisition -> logging
acquisition -> contract/httpserver/cli/cald
acquisition -> run
acquisition -> use
acquisition -> eval
```

Callers:

```text
cald/app -> acquisition
```

Adapters should not call `entry`, `observe`, `proposal`, `probe`, `promote`, or
`tracelog` directly for acquisition behavior. They should call the app layer,
which owns runtime wiring.

## Files

First V1 should keep the package small:

```text
acquisition/
  runner.go
  request.go
  state.go
  errors.go
```

`runner.go` owns the orchestration flow.

`request.go` owns public `Request`, `Result`, and narrow dependency
interfaces.

`state.go` owns the private partial-result state and trace request assembly.

`errors.go` owns acquisition-level stage error codes and coded errors.

Do not add one file per stage. The stage logic already lives in the stage
packages.

## Public Shape

Use one dependency-owning runner:

```go
type Runner struct {
	loader   ProviderLoader
	catalog  CatalogStore
	observer Observer
	proposer Proposer
	prober   Prober
	promoter Promoter
	tracer   TraceWriter
}

func NewRunner(
	loader ProviderLoader,
	catalog CatalogStore,
	observer Observer,
	proposer Proposer,
	prober Prober,
	promoter Promoter,
	tracer TraceWriter,
) *Runner

func (runner *Runner) Run(ctx context.Context, req *Request) (*Result, error)
```

`Runner` owns dependencies, so callers should use `NewRunner`. Plain request and
result values may still be assembled with literals.

`request.go` owns:

```go
type Request struct {
	ProviderID string
	Hint       string
	TraceID    string
	WorkRoot   string
}

type Result struct {
	Trace model.Trace
}
```

`ProviderID` and `WorkRoot` are required.

`Hint` is optional natural-language acquisition intent. It is trace context and
proposal relevance guidance; it must not become a hard capability-id filter.

`TraceID` is optional for `Start`; if empty, `tracelog` generates it.

## Dependency Interfaces

Use stage-specific interfaces. Do not introduce a generic `Stage` interface.

```go
type ProviderLoader interface {
	Load(context.Context, *entry.LoadRequest) (*entry.LoadResult, error)
}

type CatalogStore interface {
	ListCapabilities() ([]model.Capability, error)
}

type Observer interface {
	Observe(context.Context, *observe.Request) (*observe.Result, error)
}

type Proposer interface {
	Run(context.Context, *proposal.Request) (*proposal.Result, error)
}

type Prober interface {
	Run(context.Context, *probe.Request) (*probe.Result, error)
}

type Promoter interface {
	Run(context.Context, *promote.Request) (*promote.Result, error)
}

type TraceWriter interface {
	Start(context.Context, *tracelog.Request) (*tracelog.Result, error)
	Complete(context.Context, *tracelog.Request) (*tracelog.Result, error)
	Fail(context.Context, *tracelog.Request) (*tracelog.Result, error)
	Cancel(context.Context, *tracelog.Request) (*tracelog.Result, error)
}
```

These interfaces exist because `acquisition` is the orchestration boundary and
must be testable without live LLMs, real CLI execution, or real files. They are
not abstractions for reuse outside this package.

## Internal State

Partial results belong in a private `state` type, not in `model`:

```go
type state struct {
	TraceID      string
	StartedAt    string
	Provider     *model.Provider
	Catalog      []model.Capability
	Observations []model.Observation
	Proposal     *model.ProposalTrace
	Candidates   []model.Candidate
	ProbePlans   []proposal.ProbePlan
	Probes       []model.Probe
	Promotions   []model.Promotion
}
```

`state` exists so failure paths can preserve useful trace context. It should not
be returned directly.

## Runner Flow

`Runner.Run` should stay explicit:

```text
1. Validate runner dependencies and request.
2. Load provider with entry.
3. Start trace with tracelog.
4. Load capability catalog.
5. Observe provider.
6. Propose candidates and probe plans.
7. Probe candidates.
8. Promote passed probes.
9. Complete trace.
10. Return Result.
```

`probe.Request.WorkRoot` receives `Request.WorkRoot`. `probe` owns the
candidate-level subdirectories under that root.

`proposal.Request.TraceID`, `probe.Request.TraceID`, and trace writes must use
the same trace id.

## Failure Behavior

Provider load failure may return without writing a trace. The provider is the
first stable acquisition context, and the V1 acceptance boundary requires
failure trace behavior after provider load.

After `tracelog.Start` succeeds, every later failure should attempt to write a
terminal trace:

```text
context canceled/deadline exceeded -> tracelog.Cancel
other stage failure                -> tracelog.Fail
```

The terminal trace should include all partial results already stored in
`state`.

If terminal trace writing fails, return an acquisition error that makes the
trace write failure visible. Do not hide the original stage failure in logs.

Stage result preservation rules:

- `observe.Result` observations should be saved before returning observe
  failures when present.
- `proposal.Result.Diagnostics`, `Candidates`, and `ProbePlans` should be saved
  when a proposal error returns a partial result.
- `probe.Result.Probes` should be saved when a probe error returns a partial
  result.
- `promote.Result.Promotions` should be saved when a promotion error returns a
  partial result.

## Error Codes

`errors.go` should define acquisition-level stage codes:

```go
const (
	CodeInvalidAcquisitionInput = "invalid_acquisition_input"
	CodeProviderLoadFailed      = "provider_load_failed"
	CodeCatalogLoadFailed       = "catalog_load_failed"
	CodeObserveFailed           = "observe_failed"
	CodeProposalFailed          = "proposal_failed"
	CodeProbeFailed             = "probe_failed"
	CodePromotionFailed         = "promotion_failed"
	CodeTraceWriteFailed        = "trace_write_failed"
)
```

Trace `model.RecordError.Code` should use these acquisition-level codes. The
wrapped Go error may still preserve the lower-level package error for callers.

Do not copy lower-level package error structs into durable trace records.
