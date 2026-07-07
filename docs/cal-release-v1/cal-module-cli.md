# CAL Release V1 CLI

`cli/` owns the user-facing `calctl` command surface.

It contains command parsing, daemon lifecycle commands, local daemon client
calls, stdout/stderr rendering, and CLI error formatting. The HTTP client used by
commands lives under `cli/client/` as a CLI-internal subpackage, not as a
separate first-level module.

## Boundary

`cli/` owns:

- Cobra command construction.
- Flag and argument parsing.
- Resolving the effective `CAL_HOME` for command execution.
- Calling `cli/client` for daemon-backed commands.
- Starting the local `cald` executable in the background for daemon lifecycle
  control.
- Machine-readable JSON output and human-readable text output.
- CLI error messages and exit-code mapping.
- Logging setup for CLI commands when needed.

`cli/` does not own:

- Acquisition, run, use, eval, proposal, probe, promotion, or verification logic.
- Store, config, LLM, execution, or check internals.
- HTTP request decoding, response encoding, or route handling.
- Endpoint file shape, path policy, or writes.
- Durable record persistence.

`cli/client/` owns:

- Reading local endpoint metadata through `cald/endpoint`.
- Sending HTTP requests to the local daemon.
- Encoding `contract` request DTOs.
- Decoding `contract` response DTOs and structured errors.
- Returning typed client results to CLI commands.

`cli/client/` does not own:

- CLI stdout/stderr rendering.
- Cobra command parsing.
- CAL home resolution policy.
- Daemon start, fork, or process supervision.
- Endpoint file shape, path policy, or writes.
- Application behavior.

## Dependency Rule

```text
cli        -> contract, cli/client, logging
cli/client -> contract, cald/endpoint
```

Forbidden:

```text
cli -> acquisition
cli -> proposal
cli -> probe
cli -> promote
cli -> run
cli -> use
cli -> eval
cli -> execute
cli -> check
cli -> store
cli -> config
cli -> llm
cli -> httpserver
cli -> cald/app

cli/client -> cli
cli/client -> cald/app
cli/client -> httpserver
cli/client -> cald/daemon
cli/client -> store
cli/client -> config
cli/client -> llm
cli/client -> acquisition
cli/client -> proposal
cli/client -> probe
cli/client -> promote
cli/client -> run
cli/client -> use
cli/client -> eval
cli/client -> execute
cli/client -> check
```

`cli/client` stays under `cli` because it is an implementation detail of CLI
commands. Do not move it to a first-level `client/` package until a second real
caller appears.

## Files

Start with:

```text
cli/
  cli.go
  daemon.go
  providers.go
  acquisition.go
  capabilities.go
  runs.go
  use.go
  eval.go
  input.go
  render.go
  errors.go

  client/
    client.go
    request.go
    error.go
    client_test.go
```

Do not create separate module documentation for `cli/client`; keep its design in
this CLI document.

Do not add `traces.go` in the first version. Trace records are already produced
by acquisition/tracelog, but there is no trace contract, HTTP route, or
`cli/client` method yet. Add trace commands only after that API exists.

## CLI Data

`cli/cli.go` owns the command tree and shared command dependencies:

```go
type Options struct {
	Home    string
	Stdout  io.Writer
	Stderr  io.Writer
	Environ []string
}

type CLI struct {
	home   string
	stdout io.Writer
	stderr io.Writer
	env    []string
}

func New(opts Options) (*CLI, error)
func (cli *CLI) Command() *cobra.Command
```

Rules:

- `CLI` owns command construction. It is not the business application object;
  business behavior stays behind `cald/app` and the daemon HTTP API.
- `Home` is the already-resolved effective CAL home when provided by the
  caller. If empty, `cli` may resolve it from `CAL_HOME` or platform defaults.
- `Stdout` and `Stderr` default to process stdout/stderr when nil.
- `Environ` is optional and exists so tests can resolve environment-dependent
  behavior without mutating process-global environment.
- Use `New` because `CLI` owns output writers and command-level runtime
  dependencies.

Command implementations may use one small package-private context value to
avoid repeating setup:

```go
type commandContext struct {
	client *client.Client
}
```

Keep `commandContext` private. It is a convenience for command handlers, not a
new workflow abstraction.

## Input Parsing

`input.go` owns CLI-only input decoding:

```go
func parseInputsJSON(raw string) (map[string]any, error)
```

Rules:

- Empty input returns an empty map.
- Non-object JSON is invalid for `--inputs-json`.
- Parsed inputs are passed into `contract.RunRequest` or `contract.UseRequest`
  without business interpretation.
- Do not move execution input rendering into `cli`; that belongs to `execute`.

## CLI Client Data

`cli/client/client.go` owns the dependency-owning client:

```go
type Options struct {
	Home    string
	BaseURL string
	HTTP    *http.Client
	Timeout time.Duration
}

type Client struct {
	baseURL string
	http    *http.Client
}

func New(opts Options) (*Client, error)
```

Rules:

- `BaseURL` is optional. When present, use it directly. This is mainly for tests
  or callers that already discovered the daemon endpoint.
- `Home` is required when `BaseURL` is empty. `cli/client` reads
  `cald/endpoint` from that home.
- `cli/client` must not resolve default `CAL_HOME`; the `cli` package owns that
  policy and passes the resolved value in.
- `HTTP` is optional. If nil, create a default `http.Client`.
- `Timeout` is optional. Default it to `20m` so long-running acquisition, run,
  use, or eval requests are not cut off by the CLI transport too early.
- `Client` owns HTTP dependencies, so callers should use `New`.

## CLI Client Methods

First version methods mirror the implemented HTTP API:

```go
func (client *Client) Status(ctx context.Context) (*contract.DaemonStatus, error)
func (client *Client) Stop(ctx context.Context) (*contract.DaemonStopResponse, error)

func (client *Client) AddProvider(ctx context.Context, req *contract.AddProviderRequest) (*contract.ProviderListResponse, error)
func (client *Client) ListProviders(ctx context.Context) (*contract.ProviderListResponse, error)

func (client *Client) ListCapabilities(ctx context.Context, req *contract.CapabilityListRequest) (*contract.CapabilityListResponse, error)

func (client *Client) Acquire(ctx context.Context, req *contract.AcquisitionRequest) (*contract.AcquisitionResponse, error)
func (client *Client) AcquireStream(ctx context.Context, req *contract.AcquisitionRequest, onEvent StreamHandler) (*contract.AcquisitionResponse, error)
func (client *Client) Run(ctx context.Context, req *contract.RunRequest) (*contract.RunResponse, error)
func (client *Client) RunStream(ctx context.Context, req *contract.RunRequest, onEvent StreamHandler) (*contract.RunResponse, error)
func (client *Client) Use(ctx context.Context, req *contract.UseRequest) (*contract.UseResponse, error)
func (client *Client) UseStream(ctx context.Context, req *contract.UseRequest, onEvent StreamHandler) (*contract.UseResponse, error)
func (client *Client) Eval(ctx context.Context, req *contract.EvalRequest) (*contract.EvalResponse, error)
```

Do not add client methods for routes that do not exist yet, such as:

```text
GET /v1/providers/{id}
GET /v1/runs/{id}
GET /v1/traces/{id}
```

Those require app, contract, and HTTP server support first.

## CLI Client Routes

Route and query names are package contract strings and should be constants in
`cli/client`:

```go
const (
	routeDaemonStatus = "/v1/daemon/status"
	routeDaemonStop   = "/v1/daemon/stop"
	routeProviders    = "/v1/providers"
	routeCapabilities = "/v1/capabilities"
	routeAcquisitions = "/v1/acquisitions"
	routeAcquisitionsStream = "/v1/acquisitions/stream"
	routeRuns         = "/v1/runs"
	routeRunsStream   = "/v1/runs/stream"
	routeUses         = "/v1/uses"
	routeUsesStream   = "/v1/uses/stream"
	routeEval         = "/v1/eval"

	queryCapabilityID = "capability_id"
	queryProviderID   = "provider_id"
)
```

JSON field names inside tests may remain raw fixture strings.

## CLI Client Streaming

`cli/client/stream.go` owns SSE parsing for long-running operations. It should
not be a general-purpose event framework.

```go
type StreamEventName string

const (
	StreamEventProgress StreamEventName = "progress"
	StreamEventResult   StreamEventName = "result"
	StreamEventError    StreamEventName = "error"
)

type StreamEvent struct {
	Name StreamEventName
	Data json.RawMessage
}

type StreamHandler func(context.Context, *StreamEvent) error
```

Rules:

- Stream methods POST the same request DTOs to the matching `/stream` route.
- Send `Accept: text/event-stream`.
- Decode `progress` events and pass them to the callback.
- Decode `result` as the typed final response and return it.
- Decode `error` as `contract.ErrorResponse` and return `client.Error`.
- A callback error should stop the stream and return that error.
- The parser should support only single-line `event:` and `data:` records in
  the first version because the server controls its own JSON shape.

## CLI Client Errors

`cli/client/error.go` owns transport-level client errors:

```go
type Error struct {
	StatusCode int
	Code       contract.ErrorCode
	Message    string
}

func (err *Error) Error() string
```

Mapping:

```text
endpoint missing      -> cald_unavailable
endpoint invalid      -> cald_unavailable
HTTP dial failure     -> cald_unavailable
HTTP 4xx/5xx JSON err -> contract.ErrorResponse
HTTP 4xx/5xx non-JSON -> internal_error
success decode failed -> ordinary decode error
```

`cli` should not parse HTTP status codes directly. It should inspect
`client.Error.Code` when choosing output and exit behavior.

## Run Response Contract

`cli/client.Run` decodes a small `contract.RunResponse`. `cald/app` and
`httpserver` return the same wrapper:

```go
type RunResponse struct {
	Run *model.Run `json:"run,omitempty"`
}
```

Without this wrapper, `cli/client` would need to import `model` directly to
decode run responses. That would break the intended dependency boundary:

```text
cli/client -> contract, cald/endpoint
```

## Rendering

`render.go` owns stdout rendering only:

```go
type RenderMode string

const (
	RenderText RenderMode = "text"
	RenderJSON RenderMode = "json"
)

type RenderOptions struct {
	Mode RenderMode
}
```

Rules:

- JSON output is the stable machine-readable output.
- Human text output may be minimal in the first version.
- Structured logs must not be mixed into JSON stdout.
- Rendering should accept already-built `contract` responses; it must not call
  workflow packages or inspect store files.
- Streaming text output is progress-only plus the final summary.
- Streaming JSON output is JSON Lines with one envelope per SSE event.

## CLI Errors

`errors.go` owns CLI exit behavior:

```go
type ExitError struct {
	Code int
	Err  error
}

func (err *ExitError) Error() string
func (err *ExitError) Unwrap() error
```

Rules:

- `contract.ErrorInvalidRequest` maps to usage/input failure.
- `contract.ErrorCaldUnavailable` maps to daemon unavailable.
- Other daemon errors map to command failure.
- Command handlers should return errors; the top-level command execution path
  renders them and chooses the exit code.

## CLI Commands

The first CLI command surface is intentionally limited to routes that already
exist in `contract`, `httpserver`, and `cli/client`:

```text
calctl daemon start
calctl daemon stop
calctl daemon status --json
calctl providers add --provider-path <path> --json
calctl acquisition run --provider-id <id> --json
calctl acquisition run --provider-id <id> --hint <intent> --json
calctl acquisition run --provider-id <id> --proposal-path <path> --json
calctl acquisition run --provider-id <id> --stream
calctl acquisition run --provider-id <id> --stream --json
calctl capabilities list --json
calctl runs create --capability-id <id> --inputs-json <json> --json
calctl runs create --capability-id <id> --inputs-json <json> --min-verify-level L2 --json
calctl runs create --capability-id <id> --inputs-json <json> --stream
calctl use <intent> --json
calctl use <intent> --min-verify-level L2 --json
calctl use <intent> --stream
calctl eval --json
```

`calctl daemon start` starts the local `cald serve` process in the background,
waits until daemon status is available, and then returns the same
`contract.DaemonStatus` shape used by `calctl daemon status`.

`cald serve` remains the foreground service command for debugging and external
process managers. Do not add restart loops, pid locks, or long-running
supervision behavior in `cli`.

CLI commands should stay thin:

```text
parse flags/args
-> resolve CAL_HOME
-> call cli/client or start local cald executable
-> render response
```

Commands should not import workflow packages or inspect durable store files.

Streaming mode:

```text
--json             -> existing final JSON only
--stream           -> human-readable progress on stderr, final text on stdout
--stream --json    -> JSON Lines stream on stdout
```

JSON Lines envelope:

```json
{"event":"progress","data":{}}
{"event":"result","data":{}}
```

Deferred commands:

```text
calctl providers get --provider-path <path> --json
calctl traces get --trace-id <id> --json
calctl runs get --run-id <id> --json
```

These require matching contract DTOs, HTTP routes, app methods, and
`cli/client` methods before the CLI command is added.

## Rendering

`render.go` owns output helpers:

- JSON output.
- Table or compact text output when later required.
- Error output shape.

Machine-readable stdout must not be polluted by logs or diagnostics. Diagnostics
belong on stderr or in process logs.

## Testing

Minimum tests:

- `cli/client.New` reads `cald/endpoint`.
- `cli/client.New` accepts direct `BaseURL`.
- Missing endpoint returns `cald_unavailable`.
- HTTP structured errors decode into `client.Error`.
- HTTP dial failure returns `cald_unavailable`.
- Query methods encode `capability_id` and `provider_id`.
- POST methods encode `contract` request DTOs.
- CLI command tests verify flags map to client requests.
- CLI JSON output tests verify stable public stdout shape.

CLI tests may use fake `cli/client` dependencies at the command layer. Client
tests should use `httptest` and `cald/endpoint`; they should not import
workflow packages.
