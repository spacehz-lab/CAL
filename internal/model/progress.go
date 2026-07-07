package model

import (
	"strconv"
	"time"
)

const idPrefixProgress = "progress_"

// ProgressEvent reports live workflow progress for logs and streaming adapters.
type ProgressEvent struct {
	ID           string         `json:"id"`
	Scope        ProgressScope  `json:"scope"`
	Stage        ProgressStage  `json:"stage,omitempty"`
	Step         ProgressStep   `json:"step,omitempty"`
	Status       ProgressStatus `json:"status"`
	Message      string         `json:"message,omitempty"`
	Details      map[string]any `json:"details,omitempty"`
	TraceID      string         `json:"trace_id,omitempty"`
	RunID        string         `json:"run_id,omitempty"`
	UseID        string         `json:"use_id,omitempty"`
	ProviderID   string         `json:"provider_id,omitempty"`
	CapabilityID string         `json:"capability_id,omitempty"`
	BindingID    string         `json:"binding_id,omitempty"`
	DurationMS   int64          `json:"duration_ms,omitempty"`
	Error        *RecordError   `json:"error,omitempty"`
	CreatedAt    string         `json:"created_at,omitempty"`
}

// ProgressScope identifies the workflow that emitted one progress event.
type ProgressScope string

const (
	ProgressScopeAcquisition ProgressScope = "acquisition"
	ProgressScopeRun         ProgressScope = "run"
	ProgressScopeUse         ProgressScope = "use"
)

// ProgressStep identifies an optional substage within a broad workflow stage.
type ProgressStep string

const (
	ProgressStepProposalSurface    ProgressStep = "surface"
	ProgressStepProposalCapability ProgressStep = "capability"
	ProgressStepProposalBinding    ProgressStep = "binding"
	ProgressStepProposalEvidence   ProgressStep = "evidence"
)

// ProgressStatus identifies one live progress event state.
type ProgressStatus string

const (
	ProgressStatusStarted   ProgressStatus = "started"
	ProgressStatusSucceeded ProgressStatus = "succeeded"
	ProgressStatusFailed    ProgressStatus = "failed"
	ProgressStatusCanceled  ProgressStatus = "canceled"
)

// ProgressStage identifies a coarse user-visible workflow stage.
type ProgressStage string

const (
	ProgressStageEntry    ProgressStage = "entry"
	ProgressStageCatalog  ProgressStage = "catalog"
	ProgressStageObserve  ProgressStage = "observe"
	ProgressStageProposal ProgressStage = "proposal"
	ProgressStageProbe    ProgressStage = "probe"
	ProgressStagePromote  ProgressStage = "promote"

	ProgressStageResolve ProgressStage = "resolve"
	ProgressStageExecute ProgressStage = "execute"
	ProgressStageVerify  ProgressStage = "verify"
	ProgressStageRecord  ProgressStage = "record"

	ProgressStageSelect ProgressStage = "select"
	ProgressStagePlan   ProgressStage = "plan"
	ProgressStageRun    ProgressStage = "run"
)

// ProgressID derives a local progress event id from a timestamp.
func ProgressID(now time.Time) string {
	return idPrefixProgress + strconv.FormatInt(now.UTC().UnixNano(), 10)
}
