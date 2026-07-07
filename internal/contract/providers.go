package contract

import "github.com/spacehz-lab/cal/internal/model"

// AddProviderRequest registers one explicit provider path.
type AddProviderRequest struct {
	ProviderPath string `json:"provider_path"`
}

// ProviderListResponse returns registered providers.
type ProviderListResponse struct {
	Providers []model.Provider `json:"providers"`
}
