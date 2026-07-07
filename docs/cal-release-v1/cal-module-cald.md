# CAL Release V1 Cald

`cald/` owns the `cald` executable command layer, local daemon composition,
daemon lifecycle, and the small local endpoint file used by daemon clients.

It has four responsibilities split by package:

```text
cald          -> cald executable Cobra command surface
cald/app      -> application methods and dependency wiring
cald/daemon   -> process lifecycle and HTTP server startup
cald/endpoint -> endpoint metadata file
```

## Command

`cald` root package owns the executable command surface for `cmd/cald`.

It should stay thin:

- Build the Cobra root command.
- Resolve the effective `CAL_HOME`.
- Pass process IO and environment into daemon logging.
- Call `cald/daemon` for `serve`.

It must not import `cald/app`, `httpserver`, store, config, LLM, acquisition,
run, use, eval, proposal, probe, promotion, or verification packages.

### Command Data

Keep the first command layer small:

```go
type CommandOptions struct {
	Home    string
	Stdout  io.Writer
	Stderr  io.Writer
	Environ []string
}

func NewCommand(opts CommandOptions) (*cobra.Command, error)
```

Rules:

- `Home` is optional. If empty, resolve from `CAL_HOME` or platform defaults.
- `Stdout` and `Stderr` default to process stdout/stderr when nil.
- `Environ` defaults to `os.Environ()` when nil.
- `NewCommand` owns command construction, so `cmd/cald` should call it instead
  of assembling Cobra commands directly.
- Keep only `cald serve` in the first version. Do not add status, stop, restart,
  or background supervision commands here; those belong to `calctl` or a later
  explicitly designed daemon-management layer.

### Command Files

Start with:

```text
cald/
  command.go
  command_test.go
```

Do not create a separate `cald/command` subpackage unless the command surface
grows beyond one root file.

## App

`cald/app` owns runtime composition:

- Store, config, and LLM options.
- Proposal stage runners.
- Observation and execution drivers.
- Acquisition, run, use, and eval runners.
- Application methods that accept and return `contract` DTOs.

`cald/app` is the only adapter-side package that should import workflow
packages such as `acquisition`, `run`, `use`, and `eval`.

It is not an HTTP package, daemon lifecycle package, CLI package, endpoint-file
package, or workflow algorithm package.

### App Files

Start with a small file layout:

```text
cald/app/
  app.go
  providers.go
  capabilities.go
  acquisition.go
  runs.go
  use.go
  eval.go
  mapping.go
```

`app.go` owns `App`, `Options`, and `New`.

Feature files own the public application methods. `mapping.go` may hold small
contract/result mapping helpers. If mapping grows, split it by feature later;
do not pre-split before the code needs it.

### App Data

The first implementation should keep one dependency-owning `App`:

```go
type App struct {
	home     string
	workRoot string

	store    *store.Store
	registry *entry.Registry

	acquire *acquisition.Runner
	run     *run.Runner
	use     *use.Runner
	eval    *eval.Runner
}
```

Suggested constructor input:

```go
type Options struct {
	Home     string
	WorkRoot string
	LLM      *llm.Options
	Now      func() time.Time
}

func New(opts Options) (*App, error)
```

Rules:

- `Home` is required and is the resolved `CAL_HOME`.
- `WorkRoot` is optional. If empty, default it under `Home`, for example
  `CAL_HOME/work`.
- `LLM` is optional runtime LLM configuration.
- `Now` is optional and exists only to make trace, probe, and promotion tests
  deterministic.

### App Construction

`New` should assemble concrete dependencies in one place:

```text
store.New(Home)
store.Ensure()

config.NewFile(Home).Load()
optional LLM client

check.NewChecker()

execute/cli.NewRunner()
execute.NewRunner(cliExecutor)

entry.NewRegistry(store)

observe/cli.NewObserver()
observe.NewRunner(cli observer)

proposal.NewLiveRunner when LLM is configured

probe.NewRunner(executor, checker, probe options)
promote.NewRunner(store, now)
tracelog.NewWriter(store, now)

acquisition.NewRunner(registry, store, observer, proposer, prober, promoter, tracer)
run.NewDefaultRunner(store, executor, checker)
use.NewDefaultRunner(store, run runner, optional LLM client)
eval.NewRunner(store)
```

`cald/app` should inject the same progress logging callback into acquisition,
run, and use runners. When acquisition invokes a live proposal runner, the
acquisition context should also carry the app progress callback so proposal
substage events are logged for both blocking and streaming requests.

Progress callback behavior:

```text
blocking request -> cald/app logProgress
stream request   -> httpserver SSE handler + cald/app logProgress
```

If the `progress` package already has a request-local handler in the context,
the app callback must be composed with it rather than replacing it.

`cald/app` may read `config.json` and resolve env-based LLM secrets into
runtime `llm.Options`. Raw API keys must not be written back to config, store,
trace, logs, or contract responses.

Runtime LLM option precedence:

```text
Options.LLM
-> CAL_LLM_* environment overrides
-> config.json llm section + api_key_ref
-> no LLM client
```

Supported environment variables:

```text
CAL_LLM_API
CAL_LLM_BASE_URL
CAL_LLM_MODEL
CAL_LLM_API_KEY
```

`CAL_LLM_BASE_URL` is optional. `CAL_LLM_API`, `CAL_LLM_MODEL`, and
`CAL_LLM_API_KEY` must be present after applying config and env overrides. If a
partial `CAL_LLM_*` environment is present and the final runtime options are
incomplete, `New` should return a configuration error instead of silently
falling back.

`CAL_LLM_API_KEY` may override `config.json` `api_key_ref` for the current
process. It must never be persisted.

If LLM construction fails because no LLM is configured, `New` should still
succeed. Non-LLM methods such as provider listing, capability listing, run, use,
and eval should remain available. Live acquisition should return an explicit
LLM-not-configured error.

### App Methods

Expose application methods for `httpserver`:

```go
func (app *App) AddProvider(context.Context, *contract.AddProviderRequest) (*contract.ProviderListResponse, error)
func (app *App) ListProviders(context.Context) (*contract.ProviderListResponse, error)
func (app *App) ListCapabilities(context.Context, *contract.CapabilityListRequest) (*contract.CapabilityListResponse, error)
func (app *App) Acquire(context.Context, *contract.AcquisitionRequest) (*contract.AcquisitionResponse, error)
func (app *App) Run(context.Context, *contract.RunRequest) (*contract.RunResponse, error)
func (app *App) Use(context.Context, *contract.UseRequest) (*contract.UseResponse, error)
func (app *App) Eval(context.Context, *contract.EvalRequest) (*contract.EvalResponse, error)
```

These methods map `contract` requests into workflow requests and map workflow
results back into `contract` responses. They must not decode HTTP requests,
write HTTP responses, render CLI output, or manage daemon state.

### App Progress Logging

`cald/app` owns process-level operation and progress logs. It should consume
`model.ProgressEvent` values and write safe `slog` fields.

Log:

```text
operation start/success/failure
progress scope/stage/step/status/message
trace_id/run_id/use_id/provider_id/capability_id/binding_id
duration_ms
error_code/error
safe proposal detail scalars such as model, selected, raw_response_bytes
```

Do not log by default:

```text
API keys
prompt text
hidden model reasoning
full request inputs
command stdout/stderr
file contents
details.raw_response
full trace/run/result JSON blobs
```

`details.raw_response` may be returned to an explicit `--stream --json` caller
through SSE, but `cald/app` logs should record only safe scalar diagnostics by
default. The durable trace can still retain proposal attempts for post-run
debugging.

### Acquisition Mode

Mode dispatch is an app-level wiring concern:

```text
empty/live -> live acquisition runner
replay     -> proposal/replay runner
rules      -> proposal/rules runner
unknown    -> invalid request
```

`acquisition.Runner` must not know the public mode value. `cald/app` selects the
proposal source, then injects it into the same acquisition chain:

```text
entry -> observe -> proposer -> probe -> promote -> tracelog
```

Replay and rules must not bypass observation, probe, promotion, or trace
writing. Live mode requires an LLM client. Replay mode requires
`ProposalPath`. Rules mode requires no LLM.

### App Errors

Keep app errors small. Suggested package errors:

```go
var ErrInvalidMode = errors.New("invalid acquisition mode")
var ErrLLMNotConfigured = errors.New("llm is not configured")
var ErrProposalPathRequired = errors.New("proposal path is required")
```

`httpserver` owns mapping these errors to public `contract.ErrorResponse`
payloads and HTTP status codes.

### App Implementation Order

Implement in this order:

1. `app.go`, `providers.go`, `capabilities.go`, and `eval.go`.
2. `runs.go` and `use.go`.
3. `acquisition.go`.

This keeps the first slice usable without requiring live LLM acquisition.

## Daemon

`cald/daemon` owns local process lifecycle:

- Start local server.
- Stop daemon.
- Report daemon status.
- Publish endpoint metadata through `cald/endpoint`.
- Initialize process logging from caller-provided logging configuration.

`cald/daemon` should not contain acquisition, run, use, or eval logic.

It is the process layer that wires already-owned packages together:

```text
cald/app + httpserver + cald/endpoint + logging
```

It must not decode HTTP requests, encode HTTP responses, read or write endpoint
files manually, render CLI output, or implement workflow behavior.

### Daemon Data

Keep the first implementation small:

```go
type Options struct {
	Home            string
	WorkRoot        string
	Addr            string
	Logging         logging.Options
	ShutdownTimeout time.Duration
	Now             func() time.Time
}

type Daemon struct {
	options         Options
	shutdownTimeout time.Duration
	now             func() time.Time
}

func New(opts Options) (*Daemon, error)
func (daemon *Daemon) Serve(ctx context.Context) error
```

Rules:

- `Home` is required because it is the `CAL_HOME` used by `cald/app` and
  `cald/endpoint`.
- `WorkRoot` is optional and is passed through to `cald/app`.
- `Addr` is optional. Default it to `127.0.0.1:0`.
- `ShutdownTimeout` is optional. Default it to `3s`.
- `Now` is optional. Default it to `time.Now`.
- `Logging` is optional. If provided, call `logging.Configure` during `Serve`.
- `Daemon` owns process dependencies, so callers should use `New`.

Do not add `Manager`, `Supervisor`, pid-lock, restart, background fork, or
multi-process coordination types in the first version. CLI process spawning is a
`cli` concern, not a `cald/daemon` concern.

### Daemon Serve Flow

Main flow:

```text
1. Normalize context.
2. Configure logging.
3. Build app with app.New(app.Options{Home, WorkRoot, Now}).
4. Listen on options.Addr or 127.0.0.1:0.
5. Build contract.DaemonStatus from listener address and os.Getpid().
6. Write endpoint.Record through cald/endpoint.
7. Build httpserver.Server with app and daemon callbacks.
8. Serve HTTP.
9. Shutdown when context is canceled or /v1/daemon/stop is called.
10. Remove endpoint file before returning.
```

`Serve` should return nil when the HTTP server exits normally or with
`http.ErrServerClosed`. If the caller context is canceled, returning `ctx.Err()`
is acceptable.

### Status And Endpoint Mapping

HTTP status uses `contract.DaemonStatus`:

```go
contract.DaemonStatus{
	Running: true,
	BaseURL: "http://127.0.0.1:<port>",
	PID:     os.Getpid(),
}
```

Endpoint metadata uses `cald/endpoint.Record`:

```go
endpoint.Record{
	BaseURL:   status.BaseURL,
	PID:       status.PID,
	CreatedAt: now.Format(time.RFC3339Nano),
}
```

`cald/daemon` computes status and maps it into endpoint metadata. The endpoint
subpackage owns path policy, validation, atomic writes, reads, and removal.

### Stop And Shutdown

`cald/daemon` passes daemon callbacks into `httpserver`:

```go
httpserver.DaemonControl{
	Status: func() contract.DaemonStatus { return status },
	Stop:   func() { shutdown(...) },
}
```

Shutdown may be requested by both context cancellation and HTTP stop. Protect
shutdown with `sync.Once` so the server is not shut down twice.

Shutdown behavior:

- Use `http.Server.Shutdown` with `ShutdownTimeout`.
- Always remove the endpoint file before `Serve` returns.
- Ignore missing endpoint file during cleanup.
- Close the listener on early setup failure.

### Daemon Files

Start with:

```text
cald/daemon/
  daemon.go
  daemon_test.go
```

If `Serve` grows, split only along real responsibilities:

```text
  status.go
  shutdown.go
```

Do not pre-split before the code needs it.

### Daemon Tests

Use real local HTTP and a temporary CAL home.

Minimum tests:

- `New` rejects missing `Home`.
- `New` applies default address, shutdown timeout, and clock.
- `Serve` writes the endpoint file.
- `/v1/daemon/status` returns `Running`, `BaseURL`, and `PID`.
- `/v1/daemon/stop` triggers shutdown.
- Context cancellation triggers shutdown.
- Shutdown removes the endpoint file.
- Setup failure cleans up listener and endpoint state.

Tests should not import workflow packages directly. The closed-loop daemon test
should exercise `cald/daemon -> httpserver -> cald/app` through the local HTTP
endpoint.

## Endpoint File

The endpoint file is part of the `cald` module. It may live in a small
`cald/endpoint` code subpackage so `cald/daemon` and `cli/client` can share one
file protocol without importing each other.

`cald/endpoint` owns only the local endpoint metadata file:

- File path policy.
- JSON DTO for endpoint metadata.
- Atomic write.
- Read and missing-file behavior.
- Basic file-record validation.

It is shared by `cald/daemon` and `cli/client`.

Endpoint metadata does not belong in `contract` because it is not part of the
daemon HTTP/CLI JSON command contract.

It also does not belong in `config`: config owns durable user settings, while
the endpoint file is runtime process metadata created and removed by the daemon.

### Endpoint Data

Keep the first version intentionally small:

```go
type Record struct {
	BaseURL   string `json:"base_url"`
	PID       int    `json:"pid,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}
```

Do not add token, schema version, host, port, socket path, or process status
fields until a real caller needs them. `BaseURL` is enough for the local HTTP
client, and `PID` is diagnostic metadata.

### Endpoint Behavior

Suggested public surface:

```go
func Path(home string) string
func Read(home string) (Record, bool, error)
func Write(home string, record *Record) error
func Remove(home string) error
```

Behavior:

- `Path(home)` returns `CAL_HOME/cald/endpoint.json`.
- Empty `home` is an error for read, write, and remove.
- `Read` returns `(Record{}, false, nil)` when the endpoint file does not
  exist.
- `Write` rejects a nil record or empty `BaseURL`.
- `Write` uses `pkg/jsonfile` for atomic JSON writes.
- `Remove` ignores missing endpoint files.

### Endpoint Non-Goals

`cald/endpoint` must not own:

- Daemon start, stop, status, or process liveness checks.
- HTTP request execution.
- CLI rendering or CLI command behavior.
- `contract` request or response DTOs.
- Config loading, `CAL_HOME` resolution, logging setup, or workflow packages.

Callers pass the already resolved CAL home path into endpoint functions.

## Dependency Rule

```text
cald/app      -> contract, model, store, config, llm, acquisition, run, use, eval, entry, observe, observe/cli, proposal, probe, promote, tracelog, execute, execute/cli, check
cald/daemon   -> cald/app, cald/endpoint, httpserver, logging
cald/endpoint -> pkg/jsonfile
```

Forbidden:

```text
cald/app      -> httpserver
cald/app      -> cli
cald/app      -> cli/client
cald/app      -> cald/daemon
cald/app      -> cald/endpoint
cald/app      -> logging
cald/endpoint -> cli
cald/endpoint -> cli/client
cald/endpoint -> cald/app
cald/endpoint -> httpserver
cald/endpoint -> workflow packages
cald/daemon   -> acquisition/run/use/eval implementation logic
```
