package proposal

import (
	"log/slog"
	"sort"
	"time"

	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

const (
	logKeyTraceID                = "trace_id"
	logKeyProviderID             = "provider_id"
	logKeyModel                  = "model"
	logKeyStage                  = "stage"
	logKeyDurationMS             = "duration_ms"
	logKeyError                  = "error"
	logKeySurfaceCount           = "surface_count"
	logKeyCapabilityCount        = "capability_count"
	logKeyCandidateCount         = "candidate_count"
	logKeyProbePlanCount         = "probe_plan_count"
	logKeyConcurrency            = "concurrency"
	logKeyCompleted              = "completed"
	logKeyTotal                  = "total"
	logKeyCapabilityID           = "capability_id"
	logKeyIndex                  = "index"
	logKeyTimeoutMS              = "timeout_ms"
	logKeyCandidateIndex         = "candidate_index"
	logKeyVerifyLevel            = "verify_level"
	logKeyVerifyMethod           = "verify_method"
	logKeyRawCandidateCount      = "raw_candidate_count"
	logKeySelectedCandidateCount = "selected_candidate_count"
	logKeyKeepCount              = "keep_count"
	logKeySkipCount              = "skip_count"
	logKeyDeferCount             = "defer_count"
	logKeySkipReasons            = "skip_reasons"
	logKeyDeferReasons           = "defer_reasons"
)

type logger struct {
	providerID string
	traceID    string
}

func newLogger(providerID, traceID string) logger {
	return logger{providerID: providerID, traceID: traceID}
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

func (log logger) bindingFailed(capabilityID string, completed int64, total int, candidateCount int, started time.Time, err error, attrs ...any) {
	base := []any{
		logKeyStage, string(caltrace.ProposalStageBinding),
		logKeyCapabilityID, capabilityID,
		logKeyCompleted, completed,
		logKeyTotal, total,
		logKeyCandidateCount, candidateCount,
		logKeyDurationMS, time.Since(started).Milliseconds(),
		logKeyError, err.Error(),
	}
	log.warn("proposal binding pipeline failed", append(base, attrs...)...)
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
	args := make([]any, 0, 4+len(attrs))
	if log.traceID != "" {
		args = append(args, logKeyTraceID, log.traceID)
	}
	args = append(args, logKeyProviderID, log.providerID)
	args = append(args, attrs...)
	return args
}

func bindingStageLogAttrs(stages []caltrace.ProposalStage) []any {
	stage, ok := lastBindingStage(stages)
	if !ok {
		return nil
	}
	attrs := []any{
		logKeyRawCandidateCount, stageSummary(stage, caltrace.ProposalSummaryRaw),
		logKeySelectedCandidateCount, stageSummary(stage, caltrace.ProposalSummarySelected),
		logKeyKeepCount, stageSummary(stage, caltrace.ProposalSummaryKeep),
		logKeySkipCount, stageSummary(stage, caltrace.ProposalSummarySkip),
		logKeyDeferCount, stageSummary(stage, caltrace.ProposalSummaryDefer),
	}
	if reasons := stageDecisionReasons(stage, caltrace.ProposalDecisionSkip); len(reasons) > 0 {
		attrs = append(attrs, logKeySkipReasons, reasons)
	}
	if reasons := stageDecisionReasons(stage, caltrace.ProposalDecisionDefer); len(reasons) > 0 {
		attrs = append(attrs, logKeyDeferReasons, reasons)
	}
	return attrs
}

func lastBindingStage(stages []caltrace.ProposalStage) (caltrace.ProposalStage, bool) {
	for index := len(stages) - 1; index >= 0; index-- {
		if stages[index].Name == caltrace.ProposalStageBinding {
			return stages[index], true
		}
	}
	return caltrace.ProposalStage{}, false
}

func stageSummary(stage caltrace.ProposalStage, key caltrace.ProposalSummaryKey) int {
	if stage.Summary == nil {
		return 0
	}
	return stage.Summary[key]
}

func stageDecisionReasons(stage caltrace.ProposalStage, decision caltrace.ProposalDecision) []string {
	seen := map[string]struct{}{}
	for _, item := range stage.Items {
		if item.Decision != decision || item.Reason == "" {
			continue
		}
		seen[item.Reason] = struct{}{}
	}
	reasons := make([]string, 0, len(seen))
	for reason := range seen {
		reasons = append(reasons, reason)
	}
	sort.Strings(reasons)
	return reasons
}
