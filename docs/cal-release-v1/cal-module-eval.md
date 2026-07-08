# CAL Release V1 Eval

`eval/` owns read-only acquisition and reuse metrics.

It turns durable CAL records into stable summary metrics. It does not run
providers, select bindings, evaluate `VerifySpec`, call LLMs, or write records.

## Goal

`eval/` answers the current state of the local CAL repository:

```text
traces + runs + capabilities
-> load records from store
-> filter by optional provider/capability
-> compute acquisition metrics
-> compute reuse metrics
-> compute capability coverage metrics
-> return summary result
```

`eval` is a read-only reporting use case. It is not an acquisition, run, or use
workflow.

## Product Eval vs Benchmark Eval

V1 keeps two evaluation layers separate:

```text
internal/eval
-> product/local repository status
-> calctl eval --json and HTTP eval response

evals/cli-capability
-> executable paper benchmark
-> fixed suites, real CLIs, held-out reuse, independent oracles, reports
```

`internal/eval` must stay small and deterministic. It reports what is already in
`CAL_HOME`; it does not decide whether an arXiv benchmark case succeeded.

The arXiv/release benchmark belongs under `evals/cli-capability/`. That layer may
start `cald`, call `calctl`, run providers, invoke oracles, compare baselines,
sanitize artifacts, and render HTML. Those are experiment orchestration concerns,
not application metric concerns.

## Boundary

`eval/` owns:

- Loading trace, run, and capability records through a narrow store interface.
- Optional in-memory filtering by provider id and capability id.
- Acquisition metric aggregation from `model.Trace`.
- Reuse metric aggregation from `model.Run`.
- Capability and binding coverage aggregation from `model.Capability`.
- Stable metric result structs used by app/API adapters.

`eval/` does not own:

- Acquisition execution. That belongs to `acquisition`.
- Intent-level reuse. That belongs to `use`.
- Formal promoted capability execution. That belongs to `run`.
- Binding selection. That belongs to `run/resolve` and `use/select`.
- Provider execution. That belongs to `execute`.
- Deterministic `VerifySpec` evaluation. That belongs to `check`.
- Candidate probing or promotion. Those belong to `probe` and `promote`.
- LLM calls.
- API DTO rendering.
- Config loading, logging setup, daemon behavior, HTTP, or CLI behavior.
- Store writes or record mutation.
- Benchmark task catalogs, suites, or case definitions.
- Independent benchmark oracle execution.
- Direct CLI, LLM one-shot, or provider-tool baseline execution.
- HTML report generation or release artifact sanitization.

## Dependency Rule

```text
eval -> model, store
```

Forbidden:

```text
eval -> acquisition
eval -> proposal
eval -> probe
eval -> promote
eval -> tracelog
eval -> run
eval -> use
eval -> execute
eval -> check
eval -> llm
eval -> config
eval -> logging
eval -> contract/httpserver/cli/cald
```

`eval` should depend on an internal `Store` interface declared in `eval`, not on
a broad application interface. The concrete `store.Store` should satisfy it.

## Callers

```text
cald/app -> eval
httpserver -> cald/app -> eval
cli -> cli/client -> httpserver -> cald/app -> eval
```

CLI and HTTP code should not compute metrics directly. They should render the
result exposed by the app/API layer.

## Files

```text
eval/
  runner.go
  metrics.go
  records.go
  acquisition.go
  reuse.go
```

`runner.go` owns the public package entry point.

`metrics.go` owns shared metric structs and small counter helpers.

`records.go` owns loading a read-only snapshot from store.

`acquisition.go` owns `model.Trace` aggregation.

`reuse.go` owns `model.Run` and `model.Capability` aggregation.

Do not split provider-specific or UI-specific metric files into this package.
Those belong in adapters if they are only presentation concerns.

Benchmark-specific runner files belong under `evals/cli-capability/runner/`, not
under `internal/eval`.

## Public Shape

`runner.go` owns the request, result, store interface, and runner:

```go
type Request struct {
	ProviderID    string
	CapabilityID string
}

type Result struct {
	Acquisition AcquisitionMetrics
	Reuse       ReuseMetrics
	Capability  CapabilityMetrics
}

type Store interface {
	ListTraces() ([]model.Trace, error)
	ListRuns() ([]model.Run, error)
	ListCapabilities() ([]model.Capability, error)
}

type Runner struct {
	store Store
}

func NewRunner(store Store) *Runner

func (runner *Runner) Run(ctx context.Context, req *Request) (*Result, error)
```

Use pointer parameters and pointer return values because eval requests and
results are non-trivial workflow values. The result is still read-only from the
caller's perspective.

## Metrics

Keep the first V1 metric surface count-oriented. Avoid scores, trends,
diagnosis, benchmark judgments, and policy labels until there is a concrete
consumer that needs them.

Benchmark scores such as closed-loop success rate, oracle reuse success rate,
cost amortization, capability-model coverage, and baseline comparison belong to
`evals/cli-capability` summaries. They should not be added to `internal/eval`
unless the product API gains a concrete non-benchmark consumer for them.

### AcquisitionMetrics

`AcquisitionMetrics` is computed from `model.Trace`:

```go
type AcquisitionMetrics struct {
	Traces     CountByStatus
	Candidates int
	Probes     ProbeMetrics
	Promotions PromotionMetrics
	Errors     CountByCode
}
```

Required counts:

- Total traces.
- Running, completed, failed, and canceled traces.
- Candidate count.
- Probe count.
- Passed and failed probe count.
- Promotion count.
- Promoted capability ids.
- Promoted binding ids.
- Trace error codes.

### ReuseMetrics

`ReuseMetrics` is computed from `model.Run`:

```go
type ReuseMetrics struct {
	Runs       CountByStatus
	Verified   int
	ByProvider map[string]RunMetrics
	ByCapability map[string]RunMetrics
	Errors     CountByCode
}
```

Required counts:

- Total runs.
- Succeeded and failed runs.
- Verified runs.
- Runs grouped by provider id.
- Runs grouped by capability id.
- Run error codes.

### CapabilityMetrics

`CapabilityMetrics` is computed from `model.Capability`:

```go
type CapabilityMetrics struct {
	Capabilities int
	Bindings     int
	PromotedBindings int
	BindingsWithVerify int
	CapabilitiesWithoutBindings int
}
```

Required counts:

- Capability count.
- Binding count.
- Promoted binding count.
- Bindings with `VerifySpec`.
- Capabilities without bindings.

## Filtering

`Request.ProviderID` filters:

- traces whose `ProviderIDs`, observations, candidates, probes, or promotions
  reference the provider;
- runs with matching `ProviderID`;
- capabilities that contain at least one matching binding.

`Request.CapabilityID` filters:

- traces whose candidates or promotions reference the capability;
- runs with matching `CapabilityID`;
- the exact capability record.

Filtering stays in memory for V1. Do not push query behavior into `store` until
there is a real performance problem.

## Error Behavior

Store read errors return a Go error and no partial result.

Invalid requests return a Go error. The first V1 rules are intentionally small:

- nil request means an empty request;
- unknown provider id is not checked by `eval`;
- unknown capability id is not checked by `eval`;
- empty record sets return zero metrics.

`eval` must not convert broken persisted records into soft metric errors. Record
validation belongs to `store` and `model`; if listing records fails, `eval`
returns that failure.

## Implementation Notes

Keep the package deterministic:

- No wall-clock reads.
- No filesystem access except through `Store`.
- No goroutines.
- No LLM calls.
- No provider execution.
- No record writes.

Helper functions should stay below the main runner and metric methods in each
file. Semantic string values used for grouping, status checks, or persisted
contract fields should use typed constants from the owning `model` package.

## Tests

Add direct unit tests for:

- empty store returns zero metrics;
- trace status, probe, promotion, and error aggregation;
- run status, verified count, provider grouping, capability grouping, and error
  aggregation;
- capability/binding coverage aggregation;
- provider and capability filters;
- store read error propagation;
- context cancellation before aggregation.

Tests should use small in-memory fake stores and model constants rather than raw
semantic strings.
