# CAL Release V1 Execute

`execute/` owns provider execution and output collection.

It is shared by acquisition probing and promoted run execution. It is not a
workflow package and it does not decide whether a capability succeeded.

## Goal

`execute/` turns one selected provider execution plan plus runtime inputs into
typed outputs:

```text
Provider + Execution + inputs
-> validate executable contract
-> render runtime inputs
-> run provider adapter
-> typed outputs
```

The package should describe what the provider produced. Business success,
verification, persistence, and trace writing belong to callers.

## Boundary

`execute/` owns:

- Provider and execution compatibility checks.
- Required input detection from execution specs.
- Runtime input rendering for execution specs.
- Execution adapter dispatch by `model.ExecutionKind`.
- Typed output names and output kinds.
- CLI process invocation through `execute/cli`.
- CLI stdout, stderr, exit code, and stdout target collection.

`execute/` does not own:

- Capability or binding selection. That belongs to `run/resolve` and
  `use/select`.
- Probe workdir creation, fixture creation, or cleanup. That belongs to
  `probe`.
- Temporary target generation for missing user outputs. That belongs to
  `use/plan` or `probe`.
- VerifySpec validation or predicate evaluation. That belongs to `check`.
- Run record, trace, provider, or capability persistence. That belongs to
  `store`, `run`, `probe`, and `tracelog`.
- Logging setup, config loading, LLM calls, API transport, HTTP handlers, or
  CLI command handling.

## Dependency Rule

```text
execute     -> model
execute/cli -> model, execute
```

`execute` may use only the Go standard library plus `internal/model`.

`execute/cli` may use only the Go standard library plus `internal/model` and
`internal/execute`.

Callers:

```text
probe      -> execute
run        -> execute
use/select -> execute
```

`probe` uses `execute` to run candidate bindings before `check`.

`run` uses `execute` to run promoted bindings before optional `check`.

`use/select` may use required input inspection to compare user inputs with
candidate bindings, but it should not call `Runner.Run` directly. The normal
reuse path is:

```text
use -> run -> execute
```

`acquisition`, `cli`, `httpserver`, `cli/client`, and `cald` should not bypass their
own use-case packages to call `execute` directly.

## Files

```text
execute/
  contract.go
  runner.go
  inputs.go

  cli/
    runner.go
    command.go
```

Do not add `gui/` or `web/` packages in the first V1 slice. The result contract
must leave room for them without requiring their implementation now.

Future adapter shape:

```text
execute/gui/
  runner.go

execute/web/
  runner.go
```

## Public Shape

`contract.go` owns the runtime contract:

```go
type Request struct {
	Provider  *model.Provider
	Execution *model.Execution
	Inputs    map[string]any
}

type Result struct {
	Outputs Outputs
}

type Outputs map[OutputName]Output

type OutputName string

type Output struct {
	Kind   OutputKind
	Text   string
	Number *int
	Path   string
	MIME   string
}

type OutputKind string

type Executor interface {
	Run(context.Context, *Request) (*Result, error)
}
```

`Request`, `Result`, and `Output` are runtime-only types. They belong in
`execute`, not `model`.

Use pointer parameters and pointer return values because execution requests and
results are non-trivial runtime values.

## Output Contract

`Outputs` is a typed map. It keeps the call site close to ordinary map usage
without falling back to `map[string]any`.

Core output names:

```go
const (
	OutputStdout     OutputName = "stdout"
	OutputStderr     OutputName = "stderr"
	OutputExitCode   OutputName = "exit_code"
	OutputTarget     OutputName = "target"
	OutputText       OutputName = "text"
	OutputScreenshot OutputName = "screenshot"
	OutputURL        OutputName = "url"
	OutputDOMText    OutputName = "dom_text"
)
```

Core output kinds:

```go
const (
	OutputKindText   OutputKind = "text"
	OutputKindNumber OutputKind = "number"
	OutputKindFile   OutputKind = "file"
	OutputKindLink   OutputKind = "link"
)
```

Semantic output names and output kinds are contract strings. Define them as
typed constants in `execute` before using them in production code or contract
tests.

Adapter-specific names are allowed when a future adapter has stable semantics,
for example `window_title`, `focused_label`, or `http_status`. Those should
also be constants in the owning adapter package if they participate in
branching, validation, checking, trace output, or tests.

## Runner

`runner.go` owns adapter dispatch:

```go
type Runner struct {
	cli Executor
}

func NewRunner(cli Executor) *Runner

func (r *Runner) Run(ctx context.Context, req *Request) (*Result, error)
```

V1 should keep construction simple. A generic registry is not needed while CLI
is the only implemented adapter.

Dispatch rules:

```text
model.ExecutionKindCLI      -> cli executor
model.ExecutionKindMenu     -> unsupported
model.ExecutionKindAXAction -> unsupported
model.ExecutionKindURLOpen  -> unsupported
```

Unsupported execution kinds return an error. They do not return an empty
successful result.

## Inputs

`inputs.go` owns execution input helpers:

```go
func RequiredInputs(execution *model.Execution) ([]string, error)
func RenderArgs(execution *model.Execution, inputs map[string]any) ([]string, error)
func StdoutPathInput(execution *model.Execution) (string, bool, error)
```

Rules:

- Placeholder syntax stays `{{name}}`.
- Required input names are sorted for deterministic callers and tests.
- CLI args may be `[]string` or JSON-decoded `[]any` containing only strings.
- Missing placeholder values are errors.
- `stdout_path_input` must be a string input name, such as `target`.
- `stdout_path_input` must not be a path literal, object, array, or
  `{{placeholder}}`.

`execute` detects missing inputs. It does not invent input values.

## CLI Adapter

`execute/cli` owns CLI process behavior.

`runner.go` validates:

- `Request` is non-nil.
- `Provider` is non-nil.
- `Execution` is non-nil.
- Provider kind is `model.ProviderKindCLI`.
- Execution kind is `model.ExecutionKindCLI`.
- Provider path is non-empty.

`command.go` owns `exec.CommandContext` usage:

- Render args with `execute.RenderArgs`.
- Capture stdout and stderr.
- Capture exit code when a process starts.
- If `stdout_path_input` is present, write stdout bytes to the path found in
  `Request.Inputs`.

CLI output example:

```go
Result{
	Outputs: Outputs{
		OutputStdout: {
			Kind: OutputKindText,
			Text: "ok\n",
		},
		OutputStderr: {
			Kind: OutputKindText,
			Text: "",
		},
		OutputExitCode: {
			Kind:   OutputKindNumber,
			Number: ptr(0),
		},
		OutputTarget: {
			Kind: OutputKindFile,
			Path: "/tmp/cal/out.pdf",
			MIME: "application/pdf",
		},
	},
}
```

When stdout is written to a target path, keep stdout in `OutputStdout` and add a
file output for the target path when the input name is known.

## Error Semantics

`execute` should return Go errors for execution-layer failures:

- Invalid request.
- Unsupported execution kind.
- Provider and execution kind mismatch.
- Invalid execution spec.
- Missing runtime input.
- Command start failure.
- Context cancellation or timeout.
- Stdout target write failure.

Non-zero CLI exit code is not automatically a CAL business failure. If the
process starts and exits, return `OutputExitCode` in the result. `run`, `probe`,
and `check` decide whether that value is acceptable.

This keeps `execute` as an output collection layer instead of a verification or
business outcome layer.

## Check Integration

The first V1 implementation may adapt `execute.Result` into the existing
`check.Request` shape in `run` or `probe`:

```text
Outputs["stdout"]    -> check.Request.Stdout
Outputs["stderr"]    -> check.Request.Stderr
Outputs["exit_code"] -> check.Request.ExitCode
inputs["target"]     -> file subjects
```

Do not make `check` import `execute` in the first slice. `check` should remain
deterministic verification over model specs and primitive runtime evidence.

A later cleanup may teach `check` to read typed outputs directly if that reduces
adapter code without creating a dependency cycle or broadening `check`.

## Future GUI And Web Adapters

The typed map result lets future adapters add stable outputs without changing
the top-level `Result` struct.

GUI example:

```go
Outputs{
	OutputScreenshot: {
		Kind: OutputKindFile,
		Path: "/tmp/cal/gui-screen.png",
		MIME: "image/png",
	},
	"window_title": {
		Kind: OutputKindText,
		Text: "Export Complete",
	},
	"focused_label": {
		Kind: OutputKindText,
		Text: "Done",
	},
}
```

Web example:

```go
Outputs{
	OutputURL: {
		Kind: OutputKindLink,
		Text: "https://example.com/report",
	},
	OutputDOMText: {
		Kind: OutputKindText,
		Text: "Export ready",
	},
	OutputScreenshot: {
		Kind: OutputKindFile,
		Path: "/tmp/cal/page.png",
		MIME: "image/png",
	},
	"http_status": {
		Kind:   OutputKindNumber,
		Number: ptr(200),
	},
}
```

The adapter that creates a stable semantic output owns the constant for that
name. Common names that become cross-adapter contracts should move to
`execute/contract.go`.

## Testing

Unit tests must cover:

- Required input detection from args placeholders.
- Required input detection from `stdout_path_input`.
- Args rendering for `[]string` specs.
- Args rendering for JSON-decoded `[]any` specs.
- Missing placeholder input errors.
- Invalid args spec errors.
- Invalid `stdout_path_input` errors.
- Provider kind mismatch.
- Unsupported execution kind dispatch.
- CLI stdout and stderr capture.
- CLI exit code capture, including non-zero exit.
- Stdout target writing.
- Context cancellation.

Use local temporary executable scripts or helper test processes. Do not require
real external CLIs for unit tests.
