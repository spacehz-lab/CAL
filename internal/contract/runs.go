package contract

import "github.com/spacehz-lab/cal/internal/model"

// RunStrategy selects public binding resolution behavior.
type RunStrategy string

const (
	RunStrategyDefault RunStrategy = "default"
	RunStrategyFirst   RunStrategy = "first"
	RunStrategyBest    RunStrategy = "best"
)

// RunRequest executes one promoted capability binding.
type RunRequest struct {
	CapabilityID   string            `json:"capability_id"`
	BindingID      string            `json:"binding_id,omitempty"`
	Inputs         map[string]any    `json:"inputs"`
	ProviderID     string            `json:"provider_id,omitempty"`
	Strategy       RunStrategy       `json:"strategy,omitempty"`
	Verify         bool              `json:"verify,omitempty"`
	MinVerifyLevel model.VerifyLevel `json:"min_verify_level,omitempty"`
}

// RunResponse reports one public capability execution result.
type RunResponse struct {
	Run *model.Run `json:"run,omitempty"`
}
