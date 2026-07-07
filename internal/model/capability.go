package model

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
