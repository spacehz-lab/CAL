# CAL Release V1 Tracelog

`tracelog/` owns acquisition trace assembly and persistence.

It records what happened during one acquisition attempt. It does not run any
acquisition stage and does not decide whether candidates are valid, verified, or
promoted.

## Goal

`tracelog/` turns already-produced stage outputs into a durable `model.Trace`:

```text
Trace request
+ observations
+ proposal diagnostics
+ candidates
+ probes
+ promotions
+ optional error
-> model.Trace
-> SaveTrace
```

The normal acquisition flow is:

```text
entry -> observe -> proposal -> probe -> promote -> tracelog
```

`acquisition` owns the orchestration. `tracelog` owns only the trace record.

## Boundary

`tracelog/` owns:

- Starting a running trace.
- Completing a trace.
- Failing a trace with partial stage results.
- Canceling a trace with partial stage results.
- Trace id generation for `Start` when the caller does not provide one.
- `StartedAt` and `EndedAt` assignment.
- Assembly of `model.Trace` from `model` records.
- Saving the trace through a narrow store interface.
- Returning the saved trace shape to orchestration.

`tracelog/` does not own:

- Provider registration or loading. That belongs to `entry`.
- Provider observation. That belongs to `observe`.
- Candidate generation or prompt behavior. That belongs to `proposal`.
- Probe execution or verification classification. That belongs to `probe`.
- Promotion rules or durable capability mutation. That belongs to `promote`.
- Store paths, directory layout, JSON files, or atomic writes. That belongs to
  `store`.
- Eval aggregation. That belongs to `eval`.
- LLM calls.
- Config loading, logging setup, API transport, HTTP handlers, CLI rendering, or
  daemon lifecycle.

Trace writing must preserve partial results. A failed or canceled acquisition
should still be able to record observations, candidates, probes, promotions, and
the structured error that existed before termination.

## Dependency Rule

```text
tracelog -> model
```

`tracelog` defines a small local `Store` interface instead of importing
`store`. Runtime composition passes `*store.Store` because it satisfies the
interface.

Forbidden:

```text
tracelog -> store
tracelog -> proposal
tracelog -> probe
tracelog -> promote
tracelog -> check
tracelog -> execute
tracelog -> acquisition
tracelog -> entry
tracelog -> observe
tracelog -> run
tracelog -> use
tracelog -> eval
tracelog -> llm
tracelog -> config
tracelog -> logging
tracelog -> contract/httpserver/cli/cald
```

Callers:

```text
acquisition -> tracelog
cald/app    -> tracelog
```

## Files

First V1 should keep the package small:

```text
tracelog/
  writer.go
  request.go
  errors.go
```

`writer.go` owns the main write methods and trace assembly.

`request.go` owns `Request`, `Result`, `Store`, and time defaults.

`errors.go` owns trace-log error codes and coded errors.

## Public Shape

Use one dependency-owning writer:

```go
type Writer struct {
	store Store
	now   func() time.Time
}

func NewWriter(store Store, now func() time.Time) *Writer
func (writer *Writer) Start(ctx context.Context, req *Request) (*Result, error)
func (writer *Writer) Complete(ctx context.Context, req *Request) (*Result, error)
func (writer *Writer) Fail(ctx context.Context, req *Request) (*Result, error)
func (writer *Writer) Cancel(ctx context.Context, req *Request) (*Result, error)
```

`Writer` owns persistence, so callers should use `NewWriter`. Plain
request/result values may still be assembled with literals.

`request.go` owns:

```go
type Store interface {
	SaveTrace(trace *model.Trace) error
}

type Request struct {
	TraceID      string
	StartedAt    string
	Hint         string
	ProviderIDs  []string
	Observations []model.Observation
	Proposal     *model.ProposalTrace
	Candidates   []model.Candidate
	Probes       []model.Probe
	Promotions   []model.Promotion
	Error        *model.RecordError
}

type Result struct {
	Trace model.Trace
}
```

## Trace State Rules

`Start` writes:

```go
Status: model.TraceStatusRunning
```

If `TraceID` is empty, `Start` generates one with `model.TraceID(now)`.

Terminal writes require a `TraceID`:

```text
Complete -> TraceStatusCompleted
Fail     -> TraceStatusFailed
Cancel   -> TraceStatusCanceled
```

Terminal writes set `EndedAt`. `StartedAt` is preserved from the request; if it
is empty, the writer uses the current time.

`tracelog` does not interpret stage success. Callers decide whether to call
`Complete`, `Fail`, or `Cancel`.
