# CAL Release V1 Contract

`contract/` owns the typed JSON contract shared by `cald/app`, HTTP handlers,
the CLI daemon client, and CLI command code.

It is a transport contract package, not a service package.

## Goal

`contract/` keeps request, response, and error DTOs in one place:

```text
cli/client + httpserver + cald/app
-> shared request and response types
-> stable JSON contract
```

The package exists to prevent handlers, clients, commands, and app methods from
each inventing their own JSON structs.

## Boundary

`contract/` owns:

- Request DTOs for daemon-facing commands.
- Response DTOs for daemon HTTP and local client calls.
- Query/filter DTOs when they are part of the daemon contract.
- Error DTOs and stable error code constants.
- Response wrappers that are not durable records.
- Public string constants that appear in HTTP or CLI JSON and participate in
  branching, such as modes, strategies, and error codes.

`contract/` does not own:

- Business execution.
- Request validation that requires store, config, LLM, or workflow state.
- Acquisition, proposal, probe, promotion, run, use, or eval logic.
- HTTP routing, request decoding, or response writing.
- HTTP client behavior.
- CLI rendering.
- Store records.
- Prompt contracts.
- Local endpoint file behavior.

Endpoint file DTOs belong to `cald/endpoint`, not `contract`.

## Admission Rule

Only types visible at the daemon HTTP or CLI JSON boundary may enter
`contract`.

Do not move a type into `contract` merely because two packages need to import it.
Use the owning package instead:

```text
durable record                 -> model
workflow request/result/state   -> owning workflow package
HTTP/CLI JSON request/response  -> contract
endpoint metadata file          -> cald/endpoint
HTTP transport behavior         -> httpserver
local daemon client behavior    -> cli/client
CLI rendering                   -> cli
business mapping/orchestration  -> cald/app
```

`contract` DTOs should be plain data structs. They should not own dependencies,
perform store-backed validation, open files, call runners, or compute summaries.

## Dependency Rule

```text
contract -> model
contract -> standard library
```

Forbidden:

```text
contract -> store
contract -> config
contract -> logging
contract -> llm
contract -> acquisition
contract -> proposal
contract -> probe
contract -> promote
contract -> run
contract -> use
contract -> eval
contract -> cald/app
contract -> httpserver
contract -> cli
contract -> cli/client
```

`contract` may embed durable `model` records in responses because those records
are already stable JSON contracts.

## Files

```text
contract/
  errors.go
  providers.go
  acquisition.go
  capabilities.go
  runs.go
  use.go
  eval.go
  daemon.go
```

Do not add subpackages for V1.

## Error Contract

`errors.go` owns the common error response shape:

```go
type Error struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

type ErrorCode string

type ErrorResponse struct {
	Error Error `json:"error"`
}
```

Stable contract error codes should be constants when they cross package
boundaries or appear in CLI/HTTP contract tests.

Examples:

```go
const (
	ErrorInvalidRequest   ErrorCode = "invalid_request"
	ErrorNotFound         ErrorCode = "not_found"
	ErrorInternal         ErrorCode = "internal_error"
	ErrorCaldUnavailable ErrorCode = "cald_unavailable"
)
```

Workflow-specific error codes may live in the owning workflow package first. If
they become public HTTP/CLI contract values, mirror them into `contract`
deliberately.

## Constant Rule

Any string that is part of the public HTTP/CLI JSON contract and participates in
branching, status mapping, or contract tests must be a typed constant in
`contract`.

Examples:

```go
type AcquisitionMode string

const (
	AcquisitionModeLive   AcquisitionMode = "live"
	AcquisitionModeReplay AcquisitionMode = "replay"
	AcquisitionModeRules  AcquisitionMode = "rules"
)

type RunStrategy string

const (
	RunStrategyDefault RunStrategy = "default"
	RunStrategyFirst   RunStrategy = "first"
	RunStrategyBest    RunStrategy = "best"
)
```

Do not define constants in `contract` for purely internal workflow values unless
they are intentionally exposed through daemon HTTP or CLI JSON.

## Provider DTOs

`providers.go` owns provider command DTOs:

```go
type AddProviderRequest struct {
	ProviderPath string `json:"provider_path"`
}

type ProviderListResponse struct {
	Providers []model.Provider `json:"providers"`
}
```

Single-provider get/add responses may return `model.Provider` directly unless a
wrapper becomes necessary for compatibility.

## Acquisition DTOs

`acquisition.go` owns acquisition command DTOs:

```go
type AcquisitionRequest struct {
	ProviderID   string `json:"provider_id,omitempty"`
	Hint         string `json:"hint,omitempty"`
	ProposalPath string `json:"proposal_path,omitempty"`
	Mode         AcquisitionMode `json:"mode,omitempty"`
}

type AcquisitionResponse struct {
	TraceID              string             `json:"trace_id"`
	ProviderIDs          []string           `json:"provider_ids,omitempty"`
	CapabilitiesPromoted int                `json:"capabilities_promoted"`
	BindingsPromoted     int                `json:"bindings_promoted"`
	Trace                *model.Trace       `json:"trace,omitempty"`
	Error                *model.RecordError `json:"error,omitempty"`
}
```

Keep acquisition response fields limited to the public command/API contract.
Detailed intermediate proposal/probe state belongs in `model.Trace` and
workflow packages.

`ProposalPath` is a transport-level replay input. `contract` records the JSON
field, but does not load files or choose a proposal runner. That mapping belongs
to `cald/app`.

## Capability DTOs

`capabilities.go` owns capability list filters and read models:

```go
type CapabilityListRequest struct {
	CapabilityID string `json:"capability_id,omitempty"`
	ProviderID   string `json:"provider_id,omitempty"`
}

type CapabilityListResponse struct {
	Count        int                 `json:"count"`
	Capabilities []CapabilitySummary `json:"capabilities"`
}

type CapabilitySummary struct {
	ID          string         `json:"id"`
	Description string         `json:"description,omitempty"`
	Bindings    BindingSummary `json:"bindings"`
}

type BindingSummary struct {
	Available    int      `json:"available"`
	ProviderIDs  []string `json:"provider_ids"`
	VerifyLevels []string `json:"verify_levels"`
}
```

`contract` defines this JSON shape, but does not compute it. The calculation
belongs to the app/use-case layer.

Full capability get responses may return `model.Capability` directly.

## Run DTOs

`runs.go` owns promoted binding execution DTOs:

```go
type RunRequest struct {
	CapabilityID   string            `json:"capability_id"`
	BindingID      string            `json:"binding_id,omitempty"`
	Inputs         map[string]any    `json:"inputs"`
	ProviderID     string            `json:"provider_id,omitempty"`
	Strategy       RunStrategy       `json:"strategy,omitempty"`
	Verify         bool              `json:"verify,omitempty"`
	MinVerifyLevel model.VerifyLevel `json:"min_verify_level,omitempty"`
}

type RunResponse struct {
	Run *model.Run `json:"run,omitempty"`
}
```

Run responses use a contract wrapper so HTTP clients can decode all public
responses through the `contract` package without importing workflow packages.

## Use DTOs

`use.go` owns intent-level reuse DTOs:

```go
type UseRequest struct {
	Intent         string            `json:"intent"`
	Inputs         map[string]any    `json:"inputs,omitempty"`
	ProviderID     string            `json:"provider_id,omitempty"`
	Strategy       RunStrategy       `json:"strategy,omitempty"`
	Verify         bool              `json:"verify,omitempty"`
	MinVerifyLevel model.VerifyLevel `json:"min_verify_level,omitempty"`
}
```

`UseResponse` should contain only the public result shape needed by CLI and HTTP
clients. Selection internals stay in `use/` unless they are part of the public
contract.

## Eval DTOs

`eval.go` owns read-only metric response DTOs.

Keep eval DTOs summary-oriented. They may refer to capability ids, provider ids,
trace counts, run counts, and reuse metrics, but must not import workflow
packages.

`contract` must not import `eval`. If HTTP/CLI exposes a stable eval JSON shape,
define that public summary shape in `contract` and let `cald/app` map from
`eval.Result`.

## Response Mapping Rule

`contract` defines public response shape; `cald/app` computes and maps data into
that shape.

Examples:

- `contract.CapabilityListResponse` is computed from `model.Capability` records
  by `cald/app`, not by `contract`.
- `contract.AcquisitionResponse` is mapped from `acquisition.Result` and
  `model.Trace` by `cald/app`.
- `contract.EvalResponse` is mapped from `eval.Result` by `cald/app`.

This keeps `contract` free of workflow imports and summary logic.

## Daemon DTOs

`daemon.go` owns daemon status and stop responses:

```go
type DaemonStatus struct {
	Running bool   `json:"running"`
	BaseURL string `json:"base_url,omitempty"`
	PID     int    `json:"pid,omitempty"`
}

type DaemonStopResponse struct {
	Stopping bool `json:"stopping"`
}
```

Endpoint file DTOs do not belong in `contract`; they belong to
`cald/endpoint`.

## Callers

Expected dependency direction:

```text
cald/app   -> contract
httpserver -> contract
cli/client -> contract
cli        -> contract
```

`httpserver` decodes and encodes these DTOs.

`cli/client` uses these DTOs for typed local daemon client methods.

`cald/app` accepts contract requests, delegates to use-case packages, and
returns contract responses.

`cli` should not define separate request/response structs when a `contract` DTO
already exists.

## Anti-Bloat Rule

Do not put a type in `contract` just because multiple packages might import it.

Use this rule:

```text
model = durable record contract
contract = daemon HTTP/CLI JSON contract
workflow package = internal execution state
httpserver/cli/client/cli = transport behavior
```

If a type is not visible at the daemon HTTP or CLI JSON boundary, it should not
go in `contract`.

Before adding a new type, answer yes to at least one:

- Does this type appear directly in daemon HTTP JSON?
- Does this type appear directly in CLI JSON output?
- Is this a shared error or option string used by both HTTP and CLI contracts?

If not, keep it in `model`, the owning workflow package, `cald/app`,
`httpserver`, `cli/client`, `cli`, or `cald/endpoint`.
