# CAL Release V1 Model

`model/` owns stable CAL data contracts. It is intentionally not a workflow package.

## Boundary

`model/` contains only data structures and rules that are cross-package, persisted, or API-visible.

Put a type in `model/` only when at least one of these is true:

- It is part of a durable record.
- It is read or written by multiple business packages.
- It is a core record returned by API or CLI JSON.
- It defines stable enum strings, ID rules, or durable record validation rules.

Do not put a type in `model/` when it is only a runner request, runner result, parser output, adapter DTO, prompt output, temporary workflow state, or package-local helper shape.

Examples that belong in `model/`:

- `Provider`
- `Capability`
- `Binding`
- `Execution`
- `VerifySpec`
- `EvidenceRef`
- `Run`
- `Trace`
- `Observation`
- `Candidate`
- `Probe`
- `Promotion`
- `RecordError`
- `ProgressEvent`

Examples that do not belong in `model/`:

- `acquisition.State`
- `acquisition.Request`
- `acquisition.Result`
- `proposal/surface` parser output
- `proposal/capability` parser output
- `proposal/binding` parser output
- `proposal/evidence` parser output
- `probe.Request`
- `probe.Result`
- `execute.Request`
- `execute.Result`
- `run.Request`
- `run.Result`
- `use.Request`
- `use.Result`
- `contract.AcquisitionRequest`
- `contract.RunRequest`
- `contract.ErrorResponse`

Rule of thumb:

```text
model = stable contract
package-local type = workflow intermediate state
contract = transport request/response wrapper
```

## Compatibility Rule

Release V1 model types must be field-compatible with the current implementation.

The intended mapping is:

```text
backup/internal/core.Provider       -> internal/model.Provider
backup/internal/core.Capability     -> internal/model.Capability
backup/internal/core.Binding        -> internal/model.Binding
backup/internal/core.Execution      -> internal/model.Execution
backup/internal/core.VerifySpec     -> internal/model.VerifySpec
backup/internal/core.EvidenceRef    -> internal/model.EvidenceRef
backup/internal/core.Run            -> internal/model.Run
backup/internal/core.RecordError    -> internal/model.RecordError

backup/internal/trace.Trace         -> internal/model.Trace
backup/internal/trace.Observation   -> internal/model.Observation
backup/internal/trace.Candidate     -> internal/model.Candidate
backup/internal/trace.Probe         -> internal/model.Probe
backup/internal/trace.Promotion     -> internal/model.Promotion
backup/internal/trace.ProposalTrace -> internal/model.ProposalTrace
```

Do not change durable JSON tags, enum strings, ID rules, or validation semantics during the Release V1 rewrite.

## Files

```text
model/
  provider.go
  capability.go
  execution.go
  verify.go
  evidence.go
  run.go
  trace.go
  proposal_trace.go
  progress.go
  error.go
  ids.go
  validate.go
  verify_level.go
```

## Durable Core Records

```go
type Provider struct {
	ID      string       `json:"id"`
	Name    string       `json:"name,omitempty"`
	Kind    ProviderKind `json:"kind"`
	Path    string       `json:"path"`
	Version string       `json:"version,omitempty"`
}

type Capability struct {
	ID          string    `json:"id"`
	Description string    `json:"description,omitempty"`
	Bindings    []Binding `json:"bindings,omitempty"`
}

type Binding struct {
	ID           string        `json:"id"`
	CapabilityID string        `json:"capability_id"`
	ProviderID   string        `json:"provider_id"`
	Execution    Execution     `json:"execution"`
	Verify       *VerifySpec   `json:"verify,omitempty"`
	Evidence     []EvidenceRef `json:"evidence,omitempty"`
	State        BindingState  `json:"state"`
	CreatedAt    string        `json:"created_at,omitempty"`
}

type Run struct {
	ID           string         `json:"id"`
	CapabilityID string         `json:"capability_id"`
	BindingID    string         `json:"binding_id,omitempty"`
	ProviderID   string         `json:"provider_id,omitempty"`
	Inputs       map[string]any `json:"inputs,omitempty"`
	Outputs      map[string]any `json:"outputs,omitempty"`
	Evidence     []EvidenceRef  `json:"evidence,omitempty"`
	Status       RunStatus      `json:"status"`
	Verified     bool           `json:"verified"`
	StartedAt    string         `json:"started_at,omitempty"`
	FinishedAt   string         `json:"finished_at,omitempty"`
	DurationMS   int64          `json:"duration_ms,omitempty"`
	Error        *RecordError   `json:"error,omitempty"`
}
```

## Shared Value Objects

```go
type Execution struct {
	Kind ExecutionKind  `json:"kind"`
	Spec map[string]any `json:"spec,omitempty"`
}

type EvidenceRef struct {
	ID      string         `json:"id"`
	Type    string         `json:"type,omitempty"`
	Content map[string]any `json:"content,omitempty"`
	Ref     string         `json:"ref,omitempty"`
}

type RecordError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
```

## ProgressEvent

`model/` owns `ProgressEvent` because it is shared by workflow packages,
`cald/app` logs, HTTP SSE streaming, and CLI stream rendering. It is not a
durable proof record.

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

`Step` refines a broad stage without changing the stage enum. Release V1 uses it
for proposal live-LLM substages:

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

- `ProgressEvent` may carry safe diagnostic details for live progress.
- `Details` keys that affect tests, logs, or contracts must be stable constants
  in the owning package.
- `Details` must not carry API keys, prompt text, hidden model reasoning, full
  user inputs, command stdout/stderr, or file contents.
- `Details["raw_response"]` may be used only for explicit proposal JSON stream
  diagnostics. Process logs should prefer safe scalar fields such as model,
  selected counts, and raw response byte length.

## VerifySpec

`model/` owns the durable `VerifySpec` data shape and enum strings. It does not
own subject/predicate/param semantics or deterministic evaluation. Those belong
to `check/`.

```go
type VerifySpec struct {
	Level  VerifyLevel   `json:"level"`
	Method VerifyMethod  `json:"method"`
	Checks []VerifyCheck `json:"checks,omitempty"`
}

type VerifyCheck struct {
	Subject   VerifySubject   `json:"subject"`
	Predicate VerifyPredicate `json:"predicate"`
	Params    map[string]any  `json:"params,omitempty"`
}

type VerifySubject struct {
	Type  VerifySubjectType `json:"type"`
	Input string            `json:"input,omitempty"`
}
```

## Trace Records

Trace records are durable process artifacts. They are not the long-term capability model.

```go
type Trace struct {
	ID           string          `json:"id"`
	StartedAt    string          `json:"started_at,omitempty"`
	EndedAt      string          `json:"ended_at,omitempty"`
	Status       TraceStatus     `json:"status"`
	Hint         string          `json:"hint,omitempty"`
	ProviderIDs  []string        `json:"provider_ids,omitempty"`
	Observations []Observation   `json:"observations,omitempty"`
	Proposal     *ProposalTrace  `json:"proposal,omitempty"`
	Candidates   []Candidate     `json:"candidates,omitempty"`
	Probes       []Probe         `json:"probes,omitempty"`
	Promotions   []Promotion     `json:"promotions,omitempty"`
	Error        *RecordError    `json:"error,omitempty"`
}

type Observation struct {
	ProviderID string         `json:"provider_id"`
	Type       string         `json:"type"`
	Source     string         `json:"source,omitempty"`
	Content    map[string]any `json:"content,omitempty"`
	Error      *RecordError   `json:"error,omitempty"`
	CreatedAt  string         `json:"created_at,omitempty"`
}

type Candidate struct {
	ProviderID   string               `json:"provider_id"`
	CapabilityID string               `json:"capability_id"`
	Description  string               `json:"description,omitempty"`
	Source       string               `json:"source,omitempty"`
	Provenance   *CandidateProvenance `json:"provenance,omitempty"`
	Execution    Execution            `json:"execution"`
	CreatedAt    string               `json:"created_at,omitempty"`
}

type Probe struct {
	CandidateIndex int           `json:"candidate_index"`
	Passed         bool          `json:"passed"`
	Inputs         map[string]any `json:"inputs,omitempty"`
	Verify         VerifySpec    `json:"verify"`
	Evidence       []EvidenceRef  `json:"evidence,omitempty"`
	Reason         string        `json:"reason,omitempty"`
	Error          *RecordError  `json:"error,omitempty"`
	CreatedAt      string        `json:"created_at,omitempty"`
}

type Promotion struct {
	CandidateIndex   int    `json:"candidate_index"`
	CapabilityID     string `json:"capability_id"`
	BindingID        string `json:"binding_id,omitempty"`
	ProviderID       string `json:"provider_id"`
	CapabilityAction string `json:"capability_action,omitempty"`
	BindingAction    string `json:"binding_action,omitempty"`
	CreatedAt        string `json:"created_at,omitempty"`
}
```
