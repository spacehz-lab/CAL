package contract

import "github.com/spacehz-lab/cal/internal/model"

// UseRequest executes one intent-level reuse request.
type UseRequest struct {
	Intent         string            `json:"intent"`
	Inputs         map[string]any    `json:"inputs,omitempty"`
	ProviderID     string            `json:"provider_id,omitempty"`
	Strategy       RunStrategy       `json:"strategy,omitempty"`
	Verify         bool              `json:"verify,omitempty"`
	MinVerifyLevel model.VerifyLevel `json:"min_verify_level,omitempty"`
}

// UseResponse reports a public intent-level reuse result.
type UseResponse struct {
	ID         string             `json:"id"`
	Intent     string             `json:"intent"`
	Selection  *UseSelection      `json:"selection,omitempty"`
	Run        *model.Run         `json:"run,omitempty"`
	Status     model.RunStatus    `json:"status"`
	StartedAt  string             `json:"started_at,omitempty"`
	FinishedAt string             `json:"finished_at,omitempty"`
	DurationMS int64              `json:"duration_ms,omitempty"`
	Error      *model.RecordError `json:"error,omitempty"`
}

// UseSelection describes the selected promoted binding in public output.
type UseSelection struct {
	Source               string `json:"source,omitempty"`
	CapabilityID         string `json:"capability_id"`
	BindingID            string `json:"binding_id"`
	ProviderID           string `json:"provider_id"`
	Reason               string `json:"reason,omitempty"`
	CandidatesConsidered int    `json:"candidates_considered,omitempty"`
}
