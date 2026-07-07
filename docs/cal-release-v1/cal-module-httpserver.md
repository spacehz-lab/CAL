# CAL Release V1 HTTP Server

`httpserver/` owns HTTP server adapters for the local `cald` process.

It is transport code. It does not own application behavior, daemon lifecycle, or
local client behavior.

## Boundary

`httpserver/` owns:

- HTTP handler construction.
- Request path, method, query, and JSON body decoding.
- Calling `cald/app` methods.
- Response encoding with `contract` DTOs.
- SSE response streaming for long-running acquisition, run, and use calls.
- HTTP status mapping for structured errors.
- Daemon status and stop HTTP endpoints, using callbacks supplied by
  `cald/daemon`.

`httpserver/` does not own:

- Store, config, LLM, logging setup, or endpoint-file behavior.
- Daemon process start, stop orchestration, listen socket ownership, shutdown,
  or endpoint publication.
- Acquisition, run, use, eval, proposal, probe, promote, or trace behavior.
- CLI rendering.
- HTTP client behavior.
- Workflow progress event semantics.

`httpserver` should return an `http.Handler`. The package should not call
`ListenAndServe`, open listeners, write endpoint files, or manage process
liveness. Those belong to `cald/daemon`.

## Dependency Rule

```text
httpserver -> contract
httpserver -> progress
httpserver -> cald/app
```

Forbidden:

```text
httpserver -> store
httpserver -> config
httpserver -> llm
httpserver -> acquisition
httpserver -> proposal
httpserver -> probe
httpserver -> promote
httpserver -> run
httpserver -> use
httpserver -> eval
httpserver -> execute
httpserver -> check
httpserver -> cli
httpserver -> cli/client
httpserver -> cald/daemon
httpserver -> cald/endpoint
```

## Core Types

`server.go` owns the public construction API:

```go
type Options struct {
	App    *app.App
	Daemon DaemonControl
}

type DaemonControl struct {
	Status func() contract.DaemonStatus
	Stop   func()
}

type Server struct {
	app    *app.App
	daemon DaemonControl
	mux    *http.ServeMux
}

func New(opts Options) (*Server, error)
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request)
```

Rules:

- `App` is required.
- `DaemonControl` is optional. Missing callbacks should still allow workflow
  endpoints to work.
- `Status` should default to `contract.DaemonStatus{Running: true}` when no
  callback is provided.
- `Stop` should be called asynchronously by the stop handler when present.
- `Server` owns routing state, so callers should use `New` instead of assembling
  handlers manually.

Do not introduce an `Application` interface in the first version. There is only
one real implementation: `*cald/app.App`. If tests later become hard to write,
add a narrow interface then.

## Files

```text
httpserver/
  server.go              // Options, DaemonControl, Server, New, ServeHTTP
  response.go            // decodeJSON, writeJSON, writeError, error mapping
  sse.go                 // writeSSE, stream channel loop, event encoding
  provider_handler.go    // GET/POST providers
  capability_handler.go  // GET capabilities
  acquisition_handler.go // POST acquisitions and acquisitions/stream
  run_handler.go         // POST runs and runs/stream
  use_handler.go         // POST uses and uses/stream
  eval_handler.go        // GET eval
  daemon_handler.go      // GET status / POST stop
```

Handlers should stay thin:

```text
decode request
-> call app or daemon callback
-> encode contract response
```

No handler should contain workflow selection, store lookup, verification,
promotion, LLM, or CLI execution logic.

## Routes

First version routes:

```text
GET  /v1/daemon/status
POST /v1/daemon/stop

GET  /v1/providers
POST /v1/providers

GET  /v1/capabilities?capability_id=&provider_id=

POST /v1/acquisitions
POST /v1/acquisitions/stream
POST /v1/runs
POST /v1/runs/stream
POST /v1/uses
POST /v1/uses/stream

GET  /v1/eval?provider_id=&capability_id=
```

Do not add routes without matching `contract` DTOs and `cald/app` methods.
Specifically, do not add these in the first version:

```text
GET /v1/providers/{id}
GET /v1/runs/{id}
GET /v1/traces/{id}
```

Those require app-level methods first.

## Request Mapping

Body-based endpoints:

```text
POST /v1/providers    -> contract.AddProviderRequest
POST /v1/acquisitions -> contract.AcquisitionRequest
POST /v1/acquisitions/stream -> contract.AcquisitionRequest
POST /v1/runs         -> contract.RunRequest
POST /v1/runs/stream  -> contract.RunRequest
POST /v1/uses         -> contract.UseRequest
POST /v1/uses/stream  -> contract.UseRequest
```

Query-based endpoints:

```text
GET /v1/capabilities -> contract.CapabilityListRequest
GET /v1/eval         -> contract.EvalRequest
```

`GET /v1/providers` has no request DTO.

JSON decoding rules:

- Use `json.Decoder`.
- Reject malformed JSON.
- Reject unknown fields with `DisallowUnknownFields`.
- Treat an empty body for POST endpoints as invalid unless the app method
  explicitly accepts a nil request for that route.

## Response Mapping

Successful responses should be encoded as JSON with `Content-Type:
application/json`.

Recommended status codes:

```text
GET  success -> 200 OK
POST success -> 200 OK
```

Use `200 OK` for first version POST endpoints because the app methods return the
current command result rather than a created resource location.

SSE responses should use:

```text
Content-Type: text/event-stream
Cache-Control: no-cache
```

The handler must flush after every event.

## Streaming

Streaming handlers are transport adapters around the same `cald/app` methods as
blocking JSON handlers:

```text
decode request
-> attach request-local progress handler to context
-> run app method in a goroutine
-> write progress events
-> write one terminal result or error event
```

SSE event names:

```text
progress -> model.ProgressEvent
result   -> contract.AcquisitionResponse | contract.RunResponse | contract.UseResponse
error    -> contract.ErrorResponse
```

Rules:

- Keep blocking JSON endpoints unchanged.
- The SSE handler owns buffering, client disconnect handling, final response
  serialization, and flushing.
- Workflow runners must not import `httpserver`, `contract`, or SSE helpers.
- If the client disconnects, cancel the request context and stop writing.
- Do not stream raw command stdout, stderr, prompts, secrets, hidden model
  reasoning, or full request inputs.
- `progress` events may carry proposal diagnostic details for explicit JSON
  stream clients. `httpserver` should pass those events through without
  interpreting LLM-specific fields.
- Use typed constants for event names and stream route paths in `httpserver`.

## Error Mapping

`response.go` owns all transport error mapping.

Stable transport shape:

```go
contract.ErrorResponse{
	Error: contract.Error{
		Code:    contract.ErrorCode(...),
		Message: "...",
	},
}
```

Recommended first-version mapping:

```text
malformed JSON             -> 400 invalid_request
unknown JSON field         -> 400 invalid_request
missing required body      -> 400 invalid_request
unsupported method         -> 405 invalid_request
unknown route              -> 404 not_found
app.ErrInvalidMode         -> 400 invalid_request
app.ErrProposalPathRequired -> 400 invalid_request
app.ErrLLMNotConfigured    -> 503 cald_unavailable
context cancellation       -> 503 cald_unavailable
all other app errors       -> 500 internal_error
```

Do not duplicate error mapping inside individual handlers. Handlers should call
one shared `writeAppError` or equivalent helper.

Current `cald/app` errors are not yet all typed. Until app-level error
classification is expanded, `httpserver` should map only known exported errors
and invalid transport input precisely, then fall back to `internal_error`.

## Daemon Endpoints

`daemon_handler.go` owns only the HTTP adapter for daemon state:

```text
GET  /v1/daemon/status -> DaemonControl.Status -> contract.DaemonStatus
POST /v1/daemon/stop   -> async DaemonControl.Stop -> contract.DaemonStopResponse
```

`httpserver` must not decide whether the daemon is alive by reading pid files,
endpoint files, sockets, or process state. `cald/daemon` computes status and
passes it in.

If `Stop` is nil, the stop endpoint should return a structured
`cald_unavailable` error rather than panic.

## Testing

Use `httptest` package tests.

Minimum tests:

- `New` rejects missing app.
- `Server` implements `http.Handler`.
- Provider list/add success path.
- Capability query maps to `contract.CapabilityListRequest`.
- Run/use/acquisition POST decode valid JSON and call app.
- Eval query maps to `contract.EvalRequest`.
- Daemon status returns callback value.
- Daemon stop calls callback and returns `contract.DaemonStopResponse`.
- Acquisition/run/use stream endpoints emit progress and one terminal result on
  success.
- Stream endpoints emit one terminal error event on app failure.
- Stream endpoints stop cleanly when the client disconnects.
- Malformed JSON returns `400 invalid_request`.
- Unknown JSON fields return `400 invalid_request`.
- Unknown route returns `404 not_found`.
- Unsupported method returns `405 invalid_request`.
- `app.ErrLLMNotConfigured` maps to `503 cald_unavailable`.

Tests should not import store, config, LLM, acquisition, run, use, or eval
packages directly. Test through `cald/app` when doing an integration-style
handler test, or use narrow local test helpers only if the concrete app becomes
too expensive to construct.
