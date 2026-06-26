package eval

// Metrics is the JSON output for evaluation summaries.
type Metrics struct {
	Summary     SummaryMetrics     `json:"summary"`
	Acquisition AcquisitionMetrics `json:"acquisition,omitempty"`
	Reuse       ReuseMetrics       `json:"reuse,omitempty"`
}

// SummaryMetrics counts durable CAL records.
type SummaryMetrics struct {
	Providers        int `json:"providers"`
	Capabilities     int `json:"capabilities"`
	Bindings         int `json:"bindings"`
	PromotedBindings int `json:"promoted_bindings"`
	Traces           int `json:"traces"`
	Runs             int `json:"runs"`
}

// AcquisitionMetrics summarizes discovery and promotion evidence.
type AcquisitionMetrics struct {
	AttemptCount           int                            `json:"attempt_count,omitempty"`
	CompletedCount         int                            `json:"completed_count,omitempty"`
	FailedCount            int                            `json:"failed_count,omitempty"`
	PromotionCount         int                            `json:"promotion_count,omitempty"`
	CapabilityCreatedCount int                            `json:"capability_created_count,omitempty"`
	CapabilityReusedCount  int                            `json:"capability_reused_count,omitempty"`
	BindingCreatedCount    int                            `json:"binding_created_count,omitempty"`
	BindingUpdatedCount    int                            `json:"binding_updated_count,omitempty"`
	CandidateCount         int                            `json:"candidate_count,omitempty"`
	ProbeCount             int                            `json:"probe_count,omitempty"`
	ProbePassCount         int                            `json:"probe_pass_count,omitempty"`
	ProbeFailCount         int                            `json:"probe_fail_count,omitempty"`
	BindingPromotionRate   float64                        `json:"binding_promotion_rate,omitempty"`
	ProbeSuccessRate       float64                        `json:"probe_success_rate,omitempty"`
	ByCapability           []CapabilityAcquisitionMetrics `json:"by_capability,omitempty"`
	BySource               []SourceAcquisitionMetrics     `json:"by_source,omitempty"`
}

// CapabilityAcquisitionMetrics summarizes acquisition by requested capability.
type CapabilityAcquisitionMetrics struct {
	CapabilityID  string `json:"capability_id"`
	Attempts      int    `json:"attempts,omitempty"`
	Completed     int    `json:"completed,omitempty"`
	Failed        int    `json:"failed,omitempty"`
	Promotions    int    `json:"promotions,omitempty"`
	Candidates    int    `json:"candidates,omitempty"`
	Probes        int    `json:"probes,omitempty"`
	ProbePasses   int    `json:"probe_passes,omitempty"`
	ProbeFailures int    `json:"probe_failures,omitempty"`
}

// SourceAcquisitionMetrics summarizes acquisition by candidate source.
type SourceAcquisitionMetrics struct {
	Source        string `json:"source"`
	Attempts      int    `json:"attempts,omitempty"`
	Completed     int    `json:"completed,omitempty"`
	Failed        int    `json:"failed,omitempty"`
	Promotions    int    `json:"promotions,omitempty"`
	Candidates    int    `json:"candidates,omitempty"`
	Probes        int    `json:"probes,omitempty"`
	ProbePasses   int    `json:"probe_passes,omitempty"`
	ProbeFailures int    `json:"probe_failures,omitempty"`
}

// ReuseMetrics summarizes capability run outcomes.
type ReuseMetrics struct {
	RunCount            int     `json:"run_count,omitempty"`
	RunSuccessCount     int     `json:"run_success_count,omitempty"`
	RunFailureCount     int     `json:"run_failure_count,omitempty"`
	VerifiedRunCount    int     `json:"verified_run_count,omitempty"`
	VerifierFailCount   int     `json:"verifier_fail_count,omitempty"`
	RunSuccessRate      float64 `json:"run_success_rate,omitempty"`
	VerifiedSuccessRate float64 `json:"verified_success_rate,omitempty"`
	VerifierFailureRate float64 `json:"verifier_failure_rate,omitempty"`
	AvgRunDurationMS    int64   `json:"avg_run_duration_ms,omitempty"`
}
