package core

// ProviderKind identifies the local entry shape for a provider.
type ProviderKind string

const (
	// ProviderKindCLI identifies an executable command provider entry.
	ProviderKindCLI ProviderKind = "cli"
	// ProviderKindApp identifies an application bundle provider entry.
	ProviderKindApp ProviderKind = "app"
)

// ExecutionKind identifies the supported binding execution type.
type ExecutionKind string

const (
	// ExecutionKindCLI runs a command-line execution plan.
	ExecutionKindCLI ExecutionKind = "cli"
	// ExecutionKindMenu runs a menu-driven execution plan.
	ExecutionKindMenu ExecutionKind = "menu"
	// ExecutionKindAXAction runs an accessibility action execution plan.
	ExecutionKindAXAction ExecutionKind = "ax_action"
	// ExecutionKindURLOpen opens a URL as the execution plan.
	ExecutionKindURLOpen ExecutionKind = "url_open"
)

// BindingState identifies reusable binding lifecycle state.
type BindingState string

const (
	// BindingStatePromoted marks a verified reusable binding.
	BindingStatePromoted BindingState = "promoted"
)

// Capability is the provider-independent reusable operation record.
type Capability struct {
	ID          string    `json:"id"`
	Description string    `json:"description,omitempty"`
	Bindings    []Binding `json:"bindings,omitempty"`
}

// Provider is a discovered local provider entry.
type Provider struct {
	ID      string       `json:"id"`
	Name    string       `json:"name,omitempty"`
	Kind    ProviderKind `json:"kind"`
	Path    string       `json:"path"`
	Version string       `json:"version,omitempty"`
}

// Binding connects one capability to one provider-specific execution.
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

// Execution is the concrete provider-specific execution plan.
type Execution struct {
	Kind ExecutionKind  `json:"kind"`
	Spec map[string]any `json:"spec,omitempty"`
}

// EvidenceRef points to evidence collected during discovery or run verification.
type EvidenceRef struct {
	ID      string         `json:"id"`
	Type    string         `json:"type,omitempty"`
	Content map[string]any `json:"content,omitempty"`
	Ref     string         `json:"ref,omitempty"`
}

// Run records one capability execution attempt.
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

// RunStatus identifies run completion state.
type RunStatus string

const (
	// RunStatusSucceeded marks a successful execution.
	RunStatusSucceeded RunStatus = "succeeded"
	// RunStatusFailed marks a failed run.
	RunStatusFailed RunStatus = "failed"
)

// RecordError is the structured error shape stored in CAL records.
type RecordError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
