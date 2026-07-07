package observe

import "github.com/spacehz-lab/cal/internal/model"

// Request asks one concrete observer to inspect a provider.
type Request struct {
	Provider *model.Provider
}

// Result contains observations captured for one provider.
type Result struct {
	ProviderID   string              `json:"provider_id"`
	Observations []model.Observation `json:"observations"`
}
