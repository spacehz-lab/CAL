# CAL Release V1 Progress Streaming

This document defines how progress events reach HTTP SSE clients and CLI
streaming output. Event semantics stay in `cal-progress-events.md`.

## Goal

Long-running acquisition, run, and use calls should expose live progress without
changing the blocking JSON API:

```text
workflow runner
-> progress.Emit(ctx, event, default app logger)
-> httpserver request-local SSE handler
-> cli/client stream parser
-> cli stream renderer
```

## Non-Goals

- Do not replace acquisition traces, run records, or process logs.
- Do not stream raw stdout, stderr, prompt text, inputs, secrets, or hidden model
  reasoning.
- Do not introduce a generic workflow framework.
- Do not make CLI import workflow packages.
- Do not make workflow runners know about SSE, HTTP, or CLI rendering.

Explicit JSON streaming may include bounded proposal diagnostic details such as
the final raw LLM JSON response for one proposal substage. This is only for
debugging live acquisition and must never include prompt text, API keys, hidden
model reasoning, full user inputs, or command stdout/stderr.

## Package Boundary

`progress` owns only context-scoped fanout for `model.ProgressEvent`:

```go
type Handler func(context.Context, *model.ProgressEvent)

func WithHandler(ctx context.Context, handler Handler) context.Context
func Emit(ctx context.Context, event *model.ProgressEvent, handlers ...Handler)
```

Rules:

- `progress` imports only `model` and the standard library.
- A nil handler is valid.
- `WithHandler` should compose with an existing context handler instead of
  replacing it.
- `Emit` calls context handlers first, then explicit handlers.
- `Emit` does not return an error. Adapter backpressure and disconnect behavior
  belong to `httpserver`.
- Handlers must treat `ProgressEvent` as read-only.

Workflow runners should emit through `progress.Emit(ctx, event, runner.onProgress)`.
The explicit handler keeps existing `cald/app` operation logging. The context
handler lets one HTTP request attach an SSE observer without rebuilding the app.

When a higher-level runner invokes a nested runner that emits progress through
context only, the higher-level runner should attach its own app-level progress
handler to the nested context. For acquisition live proposal this keeps both
blocking requests and SSE requests observable:

```text
blocking request -> app log handler
stream request   -> SSE handler + app log handler
```

## HTTP SSE

Keep existing blocking endpoints:

```text
POST /v1/acquisitions
POST /v1/runs
POST /v1/uses
```

Add streaming endpoints:

```text
POST /v1/acquisitions/stream
POST /v1/runs/stream
POST /v1/uses/stream
```

All three streaming endpoints use the same request DTO as the blocking endpoint.

SSE event names:

```text
progress -> model.ProgressEvent
result   -> contract.AcquisitionResponse | contract.RunResponse | contract.UseResponse
error    -> contract.ErrorResponse
```

HTTP response headers:

```text
Content-Type: text/event-stream
Cache-Control: no-cache
```

Handler flow:

```text
decode JSON request
-> create buffered progress channel
-> attach progress handler to request context
-> run app method in goroutine
-> write progress events as they arrive
-> write exactly one terminal result or error event
-> flush after every event
```

Rules:

- The handler owns channel buffering, final event ordering, context
  cancellation, and client disconnect behavior.
- The app method still returns the authoritative final response or error.
- Progress events are best-effort live status; durable traces and run records
  remain the source of completed evidence.
- On client disconnect, cancel the request context and stop writing.
- Use typed constants for stream route paths and SSE event names in
  `httpserver`.

## CLI Client

`cli/client` adds streaming methods:

```go
func (client *Client) AcquireStream(ctx context.Context, req *contract.AcquisitionRequest, onEvent StreamHandler) (*contract.AcquisitionResponse, error)
func (client *Client) RunStream(ctx context.Context, req *contract.RunRequest, onEvent StreamHandler) (*contract.RunResponse, error)
func (client *Client) UseStream(ctx context.Context, req *contract.UseRequest, onEvent StreamHandler) (*contract.UseResponse, error)
```

`StreamHandler` receives decoded SSE envelopes before CLI rendering:

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

- Send `Accept: text/event-stream`.
- Parse `progress`, `result`, and `error` only.
- Decode `result` into the method's typed final response and return it.
- Decode `error` into `contract.ErrorResponse` and return `client.Error`.
- A callback error stops the stream.
- Keep the parser narrow: first version only needs server-controlled
  single-line `event:` and `data:` fields.

## CLI Rendering

Add `--stream` to long-running commands:

```text
calctl acquisition run --stream
calctl runs create --stream
calctl use <intent> --stream
```

Output modes:

```text
--json             -> existing final JSON only
--stream           -> human-readable progress on stderr, final text on stdout
--stream --json    -> JSON Lines event stream on stdout
```

Human text should stay compact:

```text
entry started
entry succeeded
proposal started
proposal succeeded
acquisition completed trace_...
```

JSON Lines output:

```json
{"event":"progress","data":{}}
{"event":"result","data":{}}
```

Rules:

- Do not mix logs into JSON stdout.
- Do not print raw workflow inputs, command output, prompts, secrets, or hidden
  model reasoning.
- `--stream --json` may include proposal diagnostic details in progress events,
  including `details.raw_response`, when the user explicitly requested
  streaming JSON output.
- Human `--stream` output should not print raw LLM responses. It should render
  only compact stage/step/status text.
- Final result rendering should reuse existing response renderers where
  possible.

Example proposal diagnostic JSONL:

```json
{"event":"progress","data":{"scope":"acquisition","stage":"proposal","step":"binding","status":"failed","capability_id":"text.convert","details":{"model":"deepseek-v4-pro","raw_response":"{\"candidates\":[],\"probe_material\":[]}","selected":0},"error":{"code":"proposal_stage_failed","message":"binding stage returned no candidates"}}}
```

The same event in process logs should omit `details.raw_response` and log only
safe scalar fields such as `model`, `selected`, and `raw_response_bytes`.

## Implementation Order

1. Add `internal/progress` and refactor acquisition, run, and use emitters to
   call `progress.Emit`.
2. Add acquisition SSE route and shared `httpserver` SSE writer.
3. Add `cli/client.AcquireStream` and its SSE parser.
4. Add `calctl acquisition run --stream`.
5. Add functional e2e coverage for acquisition streaming.
6. Extend the same pattern to run and use.
7. Add proposal substage progress and verify it through live LLM stream JSONL
   tests.

## Tests

Required tests:

- `progress.Emit` calls context and explicit handlers in order.
- Acquisition stream emits progress and one result on success.
- Stream endpoints emit one error event on app failure.
- Stream endpoints stop on client disconnect.
- CLI client decodes progress, result, and error events.
- CLI `--stream --json` emits valid JSON Lines.
- CLI `--stream` keeps progress separate from machine-readable JSON output.
- Live LLM e2e acquisition uses `--stream --json` and reports proposal substage
  events on failure.
