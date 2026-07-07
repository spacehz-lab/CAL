# CAL Release V1 Progress Events

Progress events describe live workflow progress for logs, HTTP SSE streams, and
CLI streaming output.

They are not a durable replacement for acquisition traces or run records.

## Goal

Expose coarse workflow progress without making callers wait silently:

```text
workflow runner
-> progress callback and context fanout
-> cald/app operation logger
-> httpserver SSE stream
-> cli streaming renderer
```

The first implementation should cover:

- Acquisition stage progress.
- Run stage progress.
- Use stage progress.
- Operation-level success and failure logs from `cald/app`.
- Proposal live-LLM substage diagnostics for debugging live acquisition.

## Boundary

Progress events own:

- Live workflow status notifications.
- Stage start, success, failure, and cancellation status.
- IDs and lightweight metadata needed to connect events to trace, run, use,
  provider, capability, and binding records.
- A stable event shape that can be serialized as SSE data.

Progress events do not own:

- Process log handler setup. That belongs to `logging`.
- Durable acquisition evidence. That belongs to `tracelog`.
- Durable run execution records. That belongs to `run` and `store`.
- HTTP streaming transport. That belongs to `httpserver`.
- CLI rendering. That belongs to `cli`.
- Stage implementation details inside `observe`, `probe`, or `promote`.
- Full inputs, outputs, stdout, stderr, prompt text, or API keys.

Proposal may emit bounded live-LLM diagnostic details because live proposal is
the longest and least deterministic acquisition stage. Those details are still
progress diagnostics, not durable proof. They must not include hidden model
reasoning, API keys, prompt text, full user inputs, or command stdout/stderr.

## Event Shape

First version should keep the type in `model` because it is shared by
acquisition, run, use, cald/app, and later contract/SSE mapping.

```go
type ProgressEvent struct {
	ID           string         `json:"id"`
	Scope        ProgressScope  `json:"scope"`
	Stage        ProgressStage  `json:"stage,omitempty"`
	Step         ProgressStep   `json:"step,omitempty"`
	Status       ProgressStatus `json:"status"`
	Message      string         `json:"message,omitempty"`
	Details      map[string]any `json:"details,omitempty"`
	TraceID      string         `json:"trace_id,omitempty"`
	RunID        string         `json:"run_id,omitempty"`
	UseID        string         `json:"use_id,omitempty"`
	ProviderID   string         `json:"provider_id,omitempty"`
	CapabilityID string         `json:"capability_id,omitempty"`
	BindingID    string         `json:"binding_id,omitempty"`
	DurationMS   int64          `json:"duration_ms,omitempty"`
	Error        *RecordError   `json:"error,omitempty"`
	CreatedAt    string         `json:"created_at,omitempty"`
}
```

Use typed constants for all public enum-like values:

```go
type ProgressScope string

const (
	ProgressScopeAcquisition ProgressScope = "acquisition"
	ProgressScopeRun         ProgressScope = "run"
	ProgressScopeUse         ProgressScope = "use"
)

type ProgressStatus string

const (
	ProgressStatusStarted   ProgressStatus = "started"
	ProgressStatusSucceeded ProgressStatus = "succeeded"
	ProgressStatusFailed    ProgressStatus = "failed"
	ProgressStatusCanceled  ProgressStatus = "canceled"
)
```

`Step` is optional and refines a broad workflow stage without changing the
stage contract. For Release V1 it is used only for proposal live-LLM substages:

```go
type ProgressStep string

const (
	ProgressStepProposalSurface    ProgressStep = "surface"
	ProgressStepProposalCapability ProgressStep = "capability"
	ProgressStepProposalBinding    ProgressStep = "binding"
	ProgressStepProposalEvidence   ProgressStep = "evidence"
)
```

Rules:

- Keep `Stage=proposal` for all proposal substage events.
- Use `Step` for `surface`, `capability`, `binding`, or `evidence`.
- Do not encode substage names as new stage strings such as
  `proposal.binding`.
- `Message` is a short safe diagnostic string.
- `Details` is for structured diagnostic metadata and must use stable keys for
  values consumed by tests, logs, or CLI output.

Stage constants should be broad and workflow-owned, not implementation-private
helper names:

```go
type ProgressStage string

const (
	ProgressStageEntry    ProgressStage = "entry"
	ProgressStageCatalog  ProgressStage = "catalog"
	ProgressStageObserve  ProgressStage = "observe"
	ProgressStageProposal ProgressStage = "proposal"
	ProgressStageProbe    ProgressStage = "probe"
	ProgressStagePromote  ProgressStage = "promote"

	ProgressStageResolve ProgressStage = "resolve"
	ProgressStageExecute ProgressStage = "execute"
	ProgressStageVerify  ProgressStage = "verify"
	ProgressStageRecord  ProgressStage = "record"

	ProgressStageSelect ProgressStage = "select"
	ProgressStagePlan   ProgressStage = "plan"
	ProgressStageRun    ProgressStage = "run"
)
```

Do not add `tracelog` as a user-facing progress stage in the first version.
Trace writing remains infrastructure for durable evidence. If trace writing
fails, report it as the acquisition operation failure through `cald/app` logs
and the final API error.

## Callback Shape

Workflow runners keep a small default callback for composition-time observers
such as `cald/app` logging:

```go
type ProgressFunc func(context.Context, *model.ProgressEvent)

type Options struct {
	OnProgress ProgressFunc
}
```

SSE needs a request-local observer, so runners should emit through the thin
`progress` package instead of calling only the default callback directly:

```go
progress.Emit(ctx, event, runner.onProgress)
```

`progress.Emit` first sends to the handler stored in `context.Context`, then to
any explicit handlers passed by the runner. This keeps request-local SSE
outside workflow constructors while preserving app-level operation logging.

Runners should treat progress callbacks as best-effort observers:

- A nil callback is valid.
- Callback failures are impossible because the function returns no error.
- Callback panics should not be recovered in the runner in the first version;
  tests and app construction should keep callbacks simple.
- Callbacks must not mutate workflow-owned records.
- Callbacks must not block on unbounded IO.

SSE buffering, backpressure, client disconnects, and serialization belong to
`httpserver`, not workflow runners.

## Acquisition Progress

`acquisition.Runner` owns acquisition stage progress because it owns the stage
order.

First version stages:

```text
entry    -> provider load
catalog  -> capability catalog load
observe  -> provider observation
proposal -> candidate and probe-plan generation
probe    -> candidate verification
promote  -> durable capability and binding promotion
```

Emit:

```text
stage started
stage succeeded
stage failed
stage canceled
```

Rules:

- `entry` started can be emitted before provider load.
- `entry` failure covers provider load failures.
- `trace_id` is available only after trace start. Early entry events may not
  have a trace id.
- After trace start, later events should include `trace_id`.
- Failed events should include `Error.Code` and `Error.Message`.
- Canceled events should use `ProgressStatusCanceled`, not failed.
- `tracelog.Start`, `tracelog.Complete`, `tracelog.Fail`, and
  `tracelog.Cancel` do not emit progress events.

The durable trace remains the source of stage evidence. Progress events are a
live status stream.

## Proposal Progress Details

`proposal.Runner` may emit finer proposal events while `acquisition.Runner`
keeps the coarse `proposal` stage event. This gives live LLM runs enough
diagnostics without making acquisition aware of proposal internals.

Release V1 proposal steps:

```text
proposal surface    -> observed surface extraction
proposal capability -> provider-independent capability planning
proposal binding    -> provider-specific candidate and probe material planning
proposal evidence   -> deterministic verify-spec planning
```

Emit:

```text
proposal step started
proposal step succeeded
proposal step failed
proposal step canceled
```

Suggested `Details` keys:

```text
proposal_stage
model
capability_id
candidate_index
raw_response
raw_response_bytes
selected
raw
keep
skip
defer
```

Rules:

- `raw_response` is the LLM's final structured output text before local parsing.
- `raw_response` is allowed only for explicit JSON streaming/debug consumers.
- Process logs should record `raw_response_bytes`, selected counts, model,
  ids, status, and error, not the full raw response.
- Do not emit hidden model reasoning or chain-of-thought.
- Do not emit prompt text.
- Do not emit API keys, full request inputs, command stdout/stderr, or file
  contents.
- Failed proposal step events should include `Error.Code` and `Error.Message`
  when a local error is available.
- Step events should include `TraceID`, `ProviderID`, and `CapabilityID` when
  known.

Proposal step progress is diagnostic. `model.ProposalTrace` remains the durable
source for proposal attempts, raw responses, summaries, and failure analysis.

## Run Progress

`run.Runner` should emit coarse run stages:

```text
resolve -> select promoted binding
execute -> execute provider command
verify  -> run deterministic verification when requested
record  -> save durable run record
```

Rules:

- `run_id` should be available once the run record is created.
- Failures that produce a durable failed run should emit failed progress and
  still save the run record.
- Store read failures before a durable run can be saved should still emit a
  failed progress event when possible.
- `record` failure should emit failed progress with `run_store_failed`.

## Use Progress

`use.Runner` should emit coarse use stages:

```text
select -> choose a promoted binding for the intent
plan   -> fill or generate execution inputs
run    -> execute selected promoted binding through run.Runner
```

Rules:

- `use_id` should be available after the use result is created.
- Selection and planning failures should emit failed events even though they do
  not create a durable run record.
- When the selected run fails, use progress should include the run error code
  and `run_id` when available.

## Cald/App Logging

`cald/app` consumes progress callbacks and writes operation logs through
`slog`.

It should log:

```text
progress event -> slog info/error
operation start
operation success
operation failure
```

Suggested fields:

```text
event
op
scope
stage
step
status
message
duration_ms
trace_id
run_id
use_id
provider_id
capability_id
binding_id
model
selected
raw_response_bytes
error_code
error
```

`cald/app` must not log:

- Raw API keys.
- Full request inputs.
- Full command stdout or stderr.
- Prompt text.
- Raw LLM responses.
- Full trace or run JSON blobs.

`cald/app` may log safe scalar values from `ProgressEvent.Details`, such as
model, selected counts, and raw response byte length. It must not log full
`raw_response` by default.

## SSE Mapping

See `cal-progress-streaming.md` for the HTTP and CLI streaming design.

Blocking JSON endpoints should remain:

```text
POST /v1/acquisitions
POST /v1/runs
POST /v1/uses
```

Streaming endpoints:

```text
POST /v1/acquisitions/stream
POST /v1/runs/stream
POST /v1/uses/stream
```

SSE event names:

```text
event: progress
data: {ProgressEvent}

event: result
data: {AcquisitionResponse|RunResponse|UseResponse}

event: error
data: {ErrorResponse}
```

The SSE adapter should own buffering, client disconnect handling, and final
response serialization. Workflow runners should only emit progress.

## Implementation Order

1. Add `model.ProgressEvent` and typed constants.
2. Add `OnProgress` to `acquisition.Runner` options and emit acquisition stage
   events.
3. Let `cald/app` inject a callback that writes progress and operation logs.
4. Add coarse progress to `run.Runner`.
5. Add coarse progress to `use.Runner`.
6. Add the `progress` context fanout package.
7. Add SSE endpoints and CLI streaming after logs and callbacks are stable.
8. Add optional proposal substage progress for live LLM diagnostics.
9. Use stream JSONL in live LLM e2e tests so failures include proposal step
   diagnostics.

## Tests

Required tests:

- Acquisition emits started and succeeded events for each successful stage.
- Acquisition emits failed stage event and terminal error code on stage failure.
- Acquisition emits canceled status when context cancellation terminates the
  flow.
- Run emits resolve, execute, verify, and record progress on verified success.
- Run emits failed progress when execution or verification fails.
- Use emits select, plan, and run progress on success.
- Use emits failed progress for selection and planning failures.
- `cald/app` logs progress without raw inputs, outputs, prompt text, or secrets.
- Proposal emits surface, capability, binding, and evidence step events during
  live proposal.
- Process logs do not include full raw LLM responses by default.

SSE tests should cover progress, result, error, and client disconnect behavior.
