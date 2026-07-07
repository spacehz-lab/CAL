# CAL Release V1 Observe

`observe/` owns provider observation contracts and provider-kind dispatch.

It turns one loaded `model.Provider` into bounded `model.Observation` records.
It does not infer capabilities, generate candidates, verify outputs, promote
bindings, or write traces.

## Goal

`observe/` is the acquisition observation boundary:

```text
model.Provider
-> select observer by provider kind
-> collect provider surface observations
-> return []model.Observation
```

The first V1 concrete observer is `observe/cli`, which captures CLI
usage surfaces. The generic `observe` package must not know CLI command
attempt details.

## Boundary

`observe/` owns:

- Observation request and result contracts.
- The `Observer` interface.
- Provider-kind dispatch.
- Generic observation constants used across acquisition.
- Input validation and unsupported provider-kind errors.
- Returning observations as `model.Observation`.

`observe/` does not own:

- Provider registration or loading. That belongs to `entry`.
- CLI help, manpage, command timeout, or text quality rules. Those belong to
  `observe/cli`.
- Provider execution for runtime capability use. That belongs to `execute`.
- Candidate generation. That belongs to `proposal`.
- Deterministic verification. That belongs to `check` and `probe`.
- Capability promotion. That belongs to `promote`.
- Acquisition trace assembly or persistence. That belongs to `tracelog`.
- LLM calls.
- Store reads or writes.
- Config loading.
- Logging setup.
- CLI, HTTP, CLI daemon client, daemon, or DTO rendering.

`observe/cli` owns:

- CLI usage collection strategy.
- `--help`, `-help`, `-h`, no-argument usage, and manual fallback attempts.
- Command timeouts for usage collection.
- Low-signal output filtering.
- Mapping usage text into `model.Observation`.

`observe/cli` does not own:

- Provider-kind dispatch outside CLI.
- Capability parsing.
- Proposal prompts.
- Verification.
- Trace writing.

## Dependency Rule

```text
observe     -> model
observe/cli -> model, observe
```

Forbidden:

```text
observe -> entry
observe -> observe/cli
observe -> proposal
observe -> probe
observe -> promote
observe -> tracelog
observe -> acquisition
observe -> execute
observe -> check
observe -> llm
observe -> store
observe -> config
observe -> logging
observe -> contract/httpserver/cli/cald
```

`observe` receives concrete observers from composition code. It must not import
`observe/cli`.

## Callers

```text
acquisition -> observe
cald/app -> observe, observe/cli
```

Production wiring should look like:

```go
observe.NewRunner(map[model.ProviderKind]observe.Observer{
	model.ProviderKindCLI: cli.NewObserver(),
})
```

Acquisition should call:

```text
entry.Load(provider_id)
-> observe.Runner.Observe(provider)
-> proposal
```

Acquisition should not call `observe/cli` directly.

## Files

```text
observe/
  contract.go
  request.go
  runner.go
  constants.go
  errors.go

  cli/
    observer.go
    usage.go
    usage_unix.go
    usage_windows.go
```

`contract.go` owns the `Observer` interface.

`request.go` owns `Request` and `Result`.

`runner.go` owns provider-kind dispatch.

`constants.go` owns generic observation type and content-key constants.

`errors.go` owns observe error codes and coded errors.

`observe/cli/observer.go` owns the CLI observer implementation.

`observe/cli/usage.go` owns CLI usage attempt order, timeout, and useful-output
filtering.

`observe/cli/usage_unix.go` owns manual fallback on non-Windows platforms.

`observe/cli/usage_windows.go` owns Windows fallback behavior.

## Public Shape

`request.go` owns:

```go
type Request struct {
	Provider *model.Provider
}

type Result struct {
	ProviderID    string
	Observations []model.Observation
}
```

`contract.go` owns:

```go
type Observer interface {
	Observe(context.Context, *Request) (*Result, error)
}
```

`runner.go` owns:

```go
type Runner struct {
	observers map[model.ProviderKind]Observer
}

func NewRunner(observers map[model.ProviderKind]Observer) *Runner

func (runner *Runner) Observe(ctx context.Context, req *Request) (*Result, error)
```

Use pointer parameters and pointer return values because provider observation is
a non-trivial workflow step.

The observer map should be copied in `NewRunner` so later caller mutation does
not change runner behavior accidentally.

## Generic Observation Contract

`constants.go` should define generic observation constants:

```go
const (
	ObservationTypeCLIOutput = "cli_output"
	ObservationContentText  = "text"
)
```

These constants are persisted in `model.Observation` records and may be read by
proposal stages. Do not use raw strings in production code for these values.

`observe/cli` may define package-local source constants:

```go
const (
	sourceHelp      = "help"
	sourceDashHelp  = "dash_help"
	sourceShortHelp = "short_help"
	sourceUsage     = "usage"
	sourceMan       = "man"
)
```

Keep source constants local unless a proposal stage needs to branch on them as
part of a stable contract.

## Runner Behavior

`Runner.Observe` flow:

```text
1. Validate runner and request.
2. Validate Provider is present.
3. Look up observer by Provider.Kind.
4. If no observer exists, return observer_not_configured.
5. Delegate to the concrete observer.
6. Normalize nil result to an observation_failed error.
7. Ensure Result.ProviderID is set.
8. Return observations.
```

`observe` should not add timestamps. Trace construction can set `CreatedAt`
when it assembles durable trace records.

## CLI Observer Behavior

`observe/cli.Observer` accepts only CLI providers:

```text
if provider.Kind != model.ProviderKindCLI:
    return empty result for provider id
```

The generic `observe.Runner` should normally prevent this path, but keeping the
concrete observer defensive is acceptable.

CLI usage attempt order:

```text
--help
-help
-h
no args
manual fallback
```

Each successful usage output becomes:

```go
model.Observation{
	ProviderID: provider.ID,
	Type:       observe.ObservationTypeCLIOutput,
	Source:     source,
	Content: map[string]any{
		observe.ObservationContentText: text,
	},
}
```

The first V1 implementation may return only the first useful usage surface. Do
not collect many large outputs until proposal actually needs them.

## Error Codes

`observe/errors.go` should define semantic constants:

```go
const (
	CodeInvalidObserveInput     = "invalid_observe_input"
	CodeUnsupportedProviderKind = "unsupported_provider_kind"
	CodeObserverNotConfigured   = "observer_not_configured"
	CodeObservationFailed       = "observation_failed"
)
```

`observe/cli` may return ordinary Go errors for command failures, timeouts, or
low-signal output. The generic runner can wrap concrete observer failures as
`observation_failed` if callers need a stable code.

## Cancellation And Timeouts

`observe.Runner` should respect context cancellation before dispatch.

`observe/cli` should apply a small command timeout to each usage command
attempt. The timeout belongs in `observe/cli`, not in generic `observe`.

Do not add goroutines or concurrent command attempts in V1.

## Tests

Add direct unit tests for generic `observe`:

- missing provider returns `invalid_observe_input`;
- unsupported provider kind returns `observer_not_configured`;
- configured observer is called for provider kind;
- observer errors are wrapped as `observation_failed`;
- nil observer result returns `observation_failed`;
- result provider id is filled when observer leaves it empty;
- context cancellation stops before dispatch.

Add direct unit tests for `observe/cli`:

- captures `--help` output as `cli_output`;
- keeps useful help text even when command exits non-zero;
- falls back from bad `--help` output to `-help` or `-h`;
- captures no-arg usage;
- rejects low-signal output;
- command timeout returns an error;
- non-CLI provider returns an empty result;
- manual fallback is covered by platform-specific tests.
