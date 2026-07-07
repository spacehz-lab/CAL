package model

// RunStatus identifies run completion state.
type RunStatus string

const (
	// RunStatusSucceeded marks a successful execution.
	RunStatusSucceeded RunStatus = "succeeded"
	// RunStatusFailed marks a failed run.
	RunStatusFailed RunStatus = "failed"
)

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
