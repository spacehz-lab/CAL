package contract

// CapabilityListRequest filters capability list responses.
type CapabilityListRequest struct {
	CapabilityID string `json:"capability_id,omitempty"`
	ProviderID   string `json:"provider_id,omitempty"`
}

// CapabilityListResponse summarizes reusable capabilities.
type CapabilityListResponse struct {
	Count        int                 `json:"count"`
	Capabilities []CapabilitySummary `json:"capabilities"`
}

// CapabilitySummary is the public compact capability list item.
type CapabilitySummary struct {
	ID          string         `json:"id"`
	Description string         `json:"description,omitempty"`
	Bindings    BindingSummary `json:"bindings"`
}

// BindingSummary summarizes promoted binding coverage for one capability.
type BindingSummary struct {
	Available    int      `json:"available"`
	ProviderIDs  []string `json:"provider_ids"`
	VerifyLevels []string `json:"verify_levels"`
}
