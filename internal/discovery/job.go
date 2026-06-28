package discovery

import (
	"strconv"
	"time"

	"github.com/spacehz-lab/cal/internal/core"
)

// JobState identifies a discovery job result state.
type JobState string

const (
	// JobStateSucceeded marks a completed discovery job.
	JobStateSucceeded JobState = "succeeded"
	// JobStateFailed marks a failed discovery job.
	JobStateFailed JobState = "failed"
)

// TargetType identifies the discovery target kind.
type TargetType string

const (
	// TargetProvider targets one known provider.
	TargetProvider TargetType = "provider"
)

// Target describes the target of one discovery job.
type Target struct {
	Type         TargetType `json:"type"`
	Value        string     `json:"value,omitempty"`
	ProviderID   string     `json:"provider_id,omitempty"`
	CapabilityID string     `json:"capability_id,omitempty"`
}

// JobResult is the stable JSON result for discovery control commands.
type JobResult struct {
	JobID                string          `json:"job_id"`
	State                JobState        `json:"state"`
	Target               Target          `json:"target"`
	ProvidersCreated     int             `json:"providers_created"`
	ProvidersUpdated     int             `json:"providers_updated"`
	CapabilitiesPromoted int             `json:"capabilities_promoted"`
	BindingsPromoted     int             `json:"bindings_promoted"`
	ProposalDurationMS   int64           `json:"proposal_duration_ms,omitempty"`
	TraceID              string          `json:"trace_id,omitempty"`
	Providers            []core.Provider `json:"providers,omitempty"`
}

func newJobID(now time.Time) string {
	return "disc_" + strconv.FormatInt(now.UTC().UnixNano(), 10)
}
