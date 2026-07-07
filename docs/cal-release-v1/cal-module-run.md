# CAL Release V1 Run

`run/` owns formal execution of a promoted capability.

It is a reuse package. It executes an already promoted binding, optionally
verifies the outputs, records a durable `model.Run`, and returns the run result.

## Goal

`run/` turns a known capability id plus runtime inputs into one durable run:

```text
capability_id + optional binding/provider selection + inputs
-> load promoted capability
-> resolve binding
-> execute provider
-> optional deterministic check
-> store Run record
```

`run` is the official execution path for promoted capabilities. It is not the
semantic intent layer and it is not acquisition probing.

## Boundary

`run/` owns:

- Formal run request validation.
- Loading promoted capabilities from store.
- Loading the selected provider from store.
- Delegating deterministic binding selection to `run/resolve`.
- Checking required execution inputs before execution.
- Calling `execute` for provider execution.
- Calling `check` when verification is requested.
- Building and saving `model.Run` records.
- Mapping run failures into `model.RecordError`.
- Returning runtime outputs and verification evidence to callers.

`run/` does not own:

- Intent understanding or semantic capability selection. That belongs to `use`
  and `use/select`.
- LLM calls.
- Proposal, probe, promote, or acquisition trace behavior.
- Candidate verification before promotion. That belongs to `probe`.
- Provider scanning or observation.
- CLI, HTTP, CLI daemon client, daemon, or DTO rendering.
- Config loading, logging setup, or environment resolution.

## Dependency Rule

```text
run         -> model, store, run/resolve, execute, check
run/resolve -> model, execute
```

Forbidden:

```text
run -> acquisition
run -> proposal
run -> probe
run -> promote
run -> tracelog
run -> use
run -> llm
run -> config
run -> logging
run -> contract/httpserver/cli/cald
```

`run/resolve` must not import `store`, `check`, or concrete execution adapters.
It receives an already loaded capability and performs deterministic selection
only.

## Callers

```text
use -> run
contract/httpserver/cald/app -> run
```

`probe` does not call `run`. Probe works on candidate bindings that are not yet
promoted:

```text
probe -> execute -> check
```

Formal reuse works through:

```text
use -> run -> execute -> check
```

## Files

```text
run/
  request.go
  runner.go
  record.go
  outputs.go

  resolve/
    runner.go
```

`outputs.go` may be omitted in the first implementation if output conversion is
small enough to stay inside `runner.go`. Add it when conversion between
`execute.Outputs`, `check.Request`, and `model.Run.Outputs` starts distracting
from the main flow.

## Public Shape

`request.go` owns the runtime request and result:

```go
type Request struct {
	CapabilityID   string
	BindingID      string
	ProviderID     string
	Inputs         map[string]any
	Verify         bool
	MinVerifyLevel model.VerifyLevel
}

type Result struct {
	Run      *model.Run
	Outputs  execute.Outputs
	Evidence []model.EvidenceRef
}
```

`Request` and `Result` are runtime-only types. They belong in `run`, not
`model`.

Use pointer parameters and pointer return values because run requests and
results are non-trivial workflow values.

## Runner

`runner.go` owns the formal run flow:

```go
type Store interface {
	GetCapability(id string) (model.Capability, bool, error)
	GetProvider(id string) (model.Provider, bool, error)
	SaveRun(run *model.Run) error
}

type Executor interface {
	Run(context.Context, *execute.Request) (*execute.Result, error)
}

type Checker interface {
	Run(context.Context, *check.Request) (*check.Result, error)
}

type Runner struct {
	store    Store
	resolver *resolve.Runner
	executor Executor
	checker  Checker
}

func NewDefaultRunner(store Store, executor Executor, checker Checker) *Runner
func NewRunner(store Store, resolver *resolve.Runner, executor Executor, checker Checker) *Runner

func (r *Runner) Run(ctx context.Context, req *Request) (*Result, error)
```

The interfaces stay in `run` because they describe what the run use case needs
from its collaborators. Do not add broad repository-wide interfaces.

`run` owns the default resolver wiring. `NewRunner` remains the narrow injection
constructor for tests or non-standard resolvers.

`cald/app` owns production wiring:

```text
store.Store
execute.NewRunner(cli.NewRunner())
check.NewChecker()
-> run.NewDefaultRunner(...)
```

## Flow

Main flow:

```text
1. Validate request.
2. Create started Run record.
3. Load capability from store.
4. Resolve binding through run/resolve.
5. Load provider from store.
6. Validate required inputs through execute.RequiredInputs.
7. Execute provider through execute.Run.
8. If Verify=true, run check against binding VerifySpec and execution outputs.
9. Convert outputs/evidence into model.Run.
10. Save Run record.
11. Return Result.
```

Every terminal path should attempt to save the run record. If saving the record
fails, return that persistence error because the durable run contract was not
met.

## Request Validation

Validation rules:

- `CapabilityID` is required.
- `Inputs` must be non-nil.
- `BindingID` is optional.
- `ProviderID` is optional.
- `Verify=false` skips deterministic verification.
- `Verify=true` requires the selected binding to declare `VerifySpec`.
- `MinVerifyLevel` is optional. When set, `run/resolve` may use it to filter
  eligible bindings.

`run` should not invent missing business inputs. Missing required inputs return
a failed run with an `invalid_run_input` error code.

## Resolve Subpackage

`run/resolve` owns deterministic binding selection for a known capability.

Public shape:

```go
type Request struct {
	Capability     *model.Capability
	BindingID      string
	ProviderID     string
	Inputs         map[string]any
	MinVerifyLevel model.VerifyLevel
}

type Result struct {
	Capability     *model.Capability
	Binding        *model.Binding
	RequiredInputs []string
}

type Runner struct{}

func NewRunner() *Runner

func (r *Runner) Run(req *Request) (*Result, error)
```

Selection rules:

- Only `model.BindingStatePromoted` bindings are eligible.
- If `BindingID` is set, only that binding is eligible.
- If `ProviderID` is set, only bindings for that provider are eligible.
- If `MinVerifyLevel` is set, bindings below that verify level are not
  eligible.
- Required inputs are computed with `execute.RequiredInputs`.
- Prefer bindings whose required inputs are satisfied by `Request.Inputs`.
- If exactly one eligible binding exists, choose it even when inputs are
  missing, so `run` can return a precise missing-input error.
- If multiple eligible bindings are satisfied, choose the first in capability
  order for V1.
- If no eligible binding exists, return a binding-not-found error.

`run/resolve` does not execute, verify, read store, or save records.

## Output Conversion

`run` receives typed outputs from `execute`:

```go
execute.Outputs
```

When verification is requested, adapt them into `check.Request`:

```text
execute.OutputStdout   -> check.Request.Stdout
execute.OutputStderr   -> check.Request.Stderr
execute.OutputExitCode -> check.Request.ExitCode
inputs                 -> check.Request.Inputs
```

`model.Run.Outputs` remains `map[string]any` because it is a durable JSON
record. Convert typed outputs into durable primitive values:

```text
text output   -> string
number output -> int
file output   -> path string
link output   -> string
```

Do not store process-specific structs in `model.Run.Outputs`.

## Success And Failure Semantics

`run` owns formal run status:

```text
model.RunStatusSucceeded
model.RunStatusFailed
```

Failure cases:

- Invalid request.
- Capability not found.
- Binding not found.
- Provider not found.
- Missing required input.
- Execute error.
- Requested verification missing from selected binding.
- Check error.
- Run record save error.

Execute non-zero exit code is not automatically an `execute` error. `run`
decides how to classify it:

- If `Verify=true`, `check` decides whether the exit code is acceptable.
- If `Verify=false`, V1 may treat a non-zero `execute.OutputExitCode` as a
  failed run to preserve ordinary CLI expectations.

When a run fails before verification, `Verified` must be false.

When `Verify=true` and check passes:

```text
Status   = succeeded
Verified = true
Evidence = check evidence
Outputs  = checked outputs merged with execution outputs
```

When `Verify=false` and execution succeeds:

```text
Status   = succeeded
Verified = false
Outputs  = execution outputs
```

## Error Codes

Use stable record error code constants in `run` before writing production code.

Initial codes:

```text
invalid_run_input
capability_not_found
binding_not_found
provider_not_found
execution_failed
verification_failed
run_store_failed
```

These codes are persisted in `model.Run.Error`. They are contract strings and
must be constants.

## Record Construction

`record.go` owns run record construction and terminal state mutation:

```go
func newRun(req *Request, now time.Time) *model.Run
func finishSucceeded(run *model.Run, started time.Time)
func finishFailed(run *model.Run, started time.Time, code, message string)
```

Run IDs may reuse the existing timestamp-based shape in the first V1 slice if
that preserves compatibility:

```text
run_<unix_nano>
```

Use UTC RFC3339Nano timestamps for `StartedAt` and `FinishedAt`.

## Testing

Unit tests should use fake store, fake executor, fake checker, and real
`run/resolve` where practical.

Tests must cover:

- Request validation.
- Capability not found.
- Binding not found.
- Provider not found.
- Missing required input.
- Successful execution without verification.
- Successful execution with verification.
- Verify requested but selected binding has no verify spec.
- Execute error saves failed run.
- Check error saves failed run.
- Non-zero exit code behavior when `Verify=false`.
- SaveRun error is returned.
- `run/resolve` filters by binding id.
- `run/resolve` filters by provider id.
- `run/resolve` ignores non-promoted bindings.
- `run/resolve` prefers satisfied inputs.
- `run/resolve` returns required inputs deterministically.

Closed-loop tests can wait until `use` or adapter packages exist.
