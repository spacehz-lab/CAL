package contract

// EvalRequest filters read-only evaluation metrics.
type EvalRequest struct {
	ProviderID   string `json:"provider_id,omitempty"`
	CapabilityID string `json:"capability_id,omitempty"`
}

// EvalResponse contains public acquisition and reuse metrics.
type EvalResponse struct {
	Acquisition AcquisitionMetrics `json:"acquisition"`
	Reuse       ReuseMetrics       `json:"reuse"`
	Capability  CapabilityMetrics  `json:"capability"`
}

// CountByStatus counts records by status.
type CountByStatus struct {
	Total  int            `json:"total"`
	ByName map[string]int `json:"by_name,omitempty"`
}

// CountByCode counts errors by public code.
type CountByCode map[string]int

// AcquisitionMetrics summarizes acquisition trace records.
type AcquisitionMetrics struct {
	Traces     CountByStatus    `json:"traces"`
	Candidates int              `json:"candidates"`
	Probes     ProbeMetrics     `json:"probes"`
	Promotions PromotionMetrics `json:"promotions"`
	Errors     CountByCode      `json:"errors,omitempty"`
}

// ProbeMetrics summarizes probe outcomes.
type ProbeMetrics struct {
	Total  int `json:"total"`
	Passed int `json:"passed"`
	Failed int `json:"failed"`
}

// PromotionMetrics summarizes promoted records from traces.
type PromotionMetrics struct {
	Total        int `json:"total"`
	Capabilities int `json:"capabilities"`
	Bindings     int `json:"bindings"`
}

// ReuseMetrics summarizes promoted capability runs.
type ReuseMetrics struct {
	Runs         CountByStatus       `json:"runs"`
	Verified     int                 `json:"verified"`
	ByProvider   map[string]RunSlice `json:"by_provider,omitempty"`
	ByCapability map[string]RunSlice `json:"by_capability,omitempty"`
	Errors       CountByCode         `json:"errors,omitempty"`
}

// RunSlice summarizes a subset of runs.
type RunSlice struct {
	Runs     CountByStatus `json:"runs"`
	Verified int           `json:"verified"`
}

// CapabilityMetrics summarizes reusable capability coverage.
type CapabilityMetrics struct {
	Capabilities                int `json:"capabilities"`
	Bindings                    int `json:"bindings"`
	PromotedBindings            int `json:"promoted_bindings"`
	BindingsWithVerify          int `json:"bindings_with_verify"`
	CapabilitiesWithoutBindings int `json:"capabilities_without_bindings"`
}
