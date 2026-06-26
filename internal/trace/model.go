package trace

import "github.com/spacehz-lab/cal/internal/core"

// Trace records what happened during one discovery attempt.
type Trace struct {
	ID           string            `json:"id"`
	StartedAt    string            `json:"started_at,omitempty"`
	EndedAt      string            `json:"ended_at,omitempty"`
	Status       Status            `json:"status"`
	Hint         string            `json:"hint,omitempty"`
	ProviderIDs  []string          `json:"provider_ids,omitempty"`
	Observations []Observation     `json:"observations,omitempty"`
	Candidates   []Candidate       `json:"candidates,omitempty"`
	Probes       []Probe           `json:"probes,omitempty"`
	Promotions   []Promotion       `json:"promotions,omitempty"`
	Error        *core.RecordError `json:"error,omitempty"`
}

// Status identifies discovery trace lifecycle state.
type Status string

const (
	// StatusRunning marks an in-progress discovery trace.
	StatusRunning Status = "running"
	// StatusCompleted marks a completed discovery trace.
	StatusCompleted Status = "completed"
	// StatusFailed marks a failed discovery trace.
	StatusFailed Status = "failed"
	// StatusCanceled marks a canceled discovery trace.
	StatusCanceled Status = "canceled"
)

// Observation records one observation captured during discovery.
type Observation struct {
	ProviderID string            `json:"provider_id"`
	Type       string            `json:"type"`
	Source     string            `json:"source,omitempty"`
	Content    map[string]any    `json:"content,omitempty"`
	Error      *core.RecordError `json:"error,omitempty"`
	CreatedAt  string            `json:"created_at,omitempty"`
}

// Candidate records one inferred candidate binding in a trace.
type Candidate struct {
	ProviderID       string               `json:"provider_id"`
	CapabilityID     string               `json:"capability_id"`
	Description      string               `json:"description,omitempty"`
	Source           string               `json:"source,omitempty"`
	Provenance       *CandidateProvenance `json:"provenance,omitempty"`
	InputConstraints map[string]any       `json:"input_constraints,omitempty"`
	Execution        core.Execution       `json:"execution"`
	Rationale        string               `json:"rationale,omitempty"`
	CreatedAt        string               `json:"created_at,omitempty"`
}

// CandidateProvenance records proposal origin without making it trusted proof.
type CandidateProvenance struct {
	Source        string `json:"source,omitempty"`
	PromptVersion string `json:"prompt_version,omitempty"`
	Model         string `json:"model,omitempty"`
	SchemaVersion string `json:"schema_version,omitempty"`
	ProposalHash  string `json:"proposal_hash,omitempty"`
}

// Probe records verification evidence for a trace candidate.
type Probe struct {
	CandidateIndex int                `json:"candidate_index"`
	Passed         bool               `json:"passed"`
	Inputs         map[string]any     `json:"inputs,omitempty"`
	Verifier       core.Verifier      `json:"verifier"`
	Evidence       []core.EvidenceRef `json:"evidence,omitempty"`
	Reason         string             `json:"reason,omitempty"`
	Error          *core.RecordError  `json:"error,omitempty"`
	CreatedAt      string             `json:"created_at,omitempty"`
}

// Promotion summarizes a promotion written from a trace.
type Promotion struct {
	CandidateIndex   int    `json:"candidate_index"`
	CapabilityID     string `json:"capability_id"`
	BindingID        string `json:"binding_id,omitempty"`
	ProviderID       string `json:"provider_id"`
	CapabilityAction string `json:"capability_action,omitempty"`
	BindingAction    string `json:"binding_action,omitempty"`
	CreatedAt        string `json:"created_at,omitempty"`
}
