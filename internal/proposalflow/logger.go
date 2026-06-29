package proposalflow

import (
	"log/slog"
	"time"

	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

const (
	logKeyProviderID      = "provider_id"
	logKeyModel           = "model"
	logKeyStage           = "stage"
	logKeyDurationMS      = "duration_ms"
	logKeyError           = "error"
	logKeySurfaceCount    = "surface_count"
	logKeyCapabilityCount = "capability_count"
	logKeyCandidateCount  = "candidate_count"
	logKeyProbePlanCount  = "probe_plan_count"
	logKeyConcurrency     = "concurrency"
	logKeyCompleted       = "completed"
	logKeyTotal           = "total"
	logKeyCapabilityID    = "capability_id"
	logKeyIndex           = "index"
	logKeyTimeoutMS       = "timeout_ms"
	logKeyCandidateIndex  = "candidate_index"
	logKeyVerifyLevel     = "verify_level"
	logKeyVerifyMethod    = "verify_method"
)

type logger struct {
	providerID string
}

func newLogger(providerID string) logger {
	return logger{providerID: providerID}
}

func (log logger) proposalStarted(model string) {
	log.info("proposal started", logKeyModel, model)
}

func (log logger) proposalCompleted(started time.Time, candidateCount, probePlanCount int) {
	log.info("proposal completed",
		logKeyDurationMS, time.Since(started).Milliseconds(),
		logKeyCandidateCount, candidateCount,
		logKeyProbePlanCount, probePlanCount,
	)
}

func (log logger) proposalFailed(stage string, started time.Time, err error) {
	log.warn("proposal failed",
		logKeyStage, stage,
		logKeyDurationMS, time.Since(started).Milliseconds(),
		logKeyError, err.Error(),
	)
}

func (log logger) stageStarted(stage caltrace.ProposalStageName, attrs ...any) {
	log.info("proposal stage started", append([]any{logKeyStage, string(stage)}, attrs...)...)
}

func (log logger) stageCompleted(stage caltrace.ProposalStageName, started time.Time, attrs ...any) {
	log.info("proposal stage completed", append([]any{
		logKeyStage, string(stage),
		logKeyDurationMS, time.Since(started).Milliseconds(),
	}, attrs...)...)
}

func (log logger) stageFailed(stage caltrace.ProposalStageName, started time.Time, err error, attrs ...any) {
	log.warn("proposal stage failed", append([]any{
		logKeyStage, string(stage),
		logKeyDurationMS, time.Since(started).Milliseconds(),
		logKeyError, err.Error(),
	}, attrs...)...)
}

func (log logger) bindingStarted(capabilityID string, index int, total int, timeout time.Duration) {
	log.info("proposal binding pipeline started",
		logKeyStage, string(caltrace.ProposalStageBinding),
		logKeyCapabilityID, capabilityID,
		logKeyIndex, index,
		logKeyTotal, total,
		logKeyTimeoutMS, timeout.Milliseconds(),
	)
}

func (log logger) bindingCompleted(capabilityID string, completed int64, total int, candidateCount int, started time.Time) {
	log.info("proposal binding pipeline completed",
		logKeyStage, string(caltrace.ProposalStageBinding),
		logKeyCapabilityID, capabilityID,
		logKeyCompleted, completed,
		logKeyTotal, total,
		logKeyCandidateCount, candidateCount,
		logKeyDurationMS, time.Since(started).Milliseconds(),
	)
}

func (log logger) bindingFailed(capabilityID string, completed int64, total int, candidateCount int, started time.Time, err error) {
	log.warn("proposal binding pipeline failed",
		logKeyStage, string(caltrace.ProposalStageBinding),
		logKeyCapabilityID, capabilityID,
		logKeyCompleted, completed,
		logKeyTotal, total,
		logKeyCandidateCount, candidateCount,
		logKeyDurationMS, time.Since(started).Milliseconds(),
		logKeyError, err.Error(),
	)
}

func (log logger) evidenceStarted(capabilityID string, candidateIndex int) {
	log.info("proposal evidence started",
		logKeyStage, string(caltrace.ProposalStageEvidence),
		logKeyCapabilityID, capabilityID,
		logKeyCandidateIndex, candidateIndex,
	)
}

func (log logger) evidenceCompleted(capabilityID string, candidateIndex int, started time.Time, verifyLevel, verifyMethod string) {
	log.info("proposal evidence completed",
		logKeyStage, string(caltrace.ProposalStageEvidence),
		logKeyCapabilityID, capabilityID,
		logKeyCandidateIndex, candidateIndex,
		logKeyDurationMS, time.Since(started).Milliseconds(),
		logKeyVerifyLevel, verifyLevel,
		logKeyVerifyMethod, verifyMethod,
	)
}

func (log logger) evidenceFailed(capabilityID string, candidateIndex int, started time.Time, err error) {
	log.warn("proposal evidence failed",
		logKeyStage, string(caltrace.ProposalStageEvidence),
		logKeyCapabilityID, capabilityID,
		logKeyCandidateIndex, candidateIndex,
		logKeyDurationMS, time.Since(started).Milliseconds(),
		logKeyError, err.Error(),
	)
}

func (log logger) info(message string, attrs ...any) {
	slog.Info(message, log.attrs(attrs...)...)
}

func (log logger) warn(message string, attrs ...any) {
	slog.Warn(message, log.attrs(attrs...)...)
}

func (log logger) attrs(attrs ...any) []any {
	args := make([]any, 0, 2+len(attrs))
	args = append(args, logKeyProviderID, log.providerID)
	args = append(args, attrs...)
	return args
}
