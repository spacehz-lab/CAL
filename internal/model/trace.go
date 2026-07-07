package model

// Trace records what happened during one discovery attempt.
type Trace struct {
	ID           string         `json:"id"`
	StartedAt    string         `json:"started_at,omitempty"`
	EndedAt      string         `json:"ended_at,omitempty"`
	Status       TraceStatus    `json:"status"`
	Hint         string         `json:"hint,omitempty"`
	ProviderIDs  []string       `json:"provider_ids,omitempty"`
	Observations []Observation  `json:"observations,omitempty"`
	Proposal     *ProposalTrace `json:"proposal,omitempty"`
	Candidates   []Candidate    `json:"candidates,omitempty"`
	Probes       []Probe        `json:"probes,omitempty"`
	Promotions   []Promotion    `json:"promotions,omitempty"`
	Error        *RecordError   `json:"error,omitempty"`
}

// TraceStatus identifies discovery trace lifecycle state.
type TraceStatus string

const (
	// TraceStatusRunning marks an in-progress discovery trace.
	TraceStatusRunning TraceStatus = "running"
	// TraceStatusCompleted marks a completed discovery trace.
	TraceStatusCompleted TraceStatus = "completed"
	// TraceStatusFailed marks a failed discovery trace.
	TraceStatusFailed TraceStatus = "failed"
	// TraceStatusCanceled marks a canceled discovery trace.
	TraceStatusCanceled TraceStatus = "canceled"
)

// Observation records one observation captured during discovery.
type Observation struct {
	ProviderID string         `json:"provider_id"`
	Type       string         `json:"type"`
	Source     string         `json:"source,omitempty"`
	Content    map[string]any `json:"content,omitempty"`
	Error      *RecordError   `json:"error,omitempty"`
	CreatedAt  string         `json:"created_at,omitempty"`
}

// Candidate records one inferred candidate binding in a trace.
type Candidate struct {
	ProviderID   string               `json:"provider_id"`
	CapabilityID string               `json:"capability_id"`
	Description  string               `json:"description,omitempty"`
	Source       string               `json:"source,omitempty"`
	Provenance   *CandidateProvenance `json:"provenance,omitempty"`
	Execution    Execution            `json:"execution"`
	CreatedAt    string               `json:"created_at,omitempty"`
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
	CandidateIndex int            `json:"candidate_index"`
	Passed         bool           `json:"passed"`
	Inputs         map[string]any `json:"inputs,omitempty"`
	Verify         VerifySpec     `json:"verify"`
	Evidence       []EvidenceRef  `json:"evidence,omitempty"`
	Reason         string         `json:"reason,omitempty"`
	Error          *RecordError   `json:"error,omitempty"`
	CreatedAt      string         `json:"created_at,omitempty"`
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
