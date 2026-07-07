package proposal

import (
	"context"
	"errors"
	"time"

	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/progress"
)

const (
	progressDetailProposalStage   = "proposal_stage"
	progressDetailModel           = "model"
	progressDetailCandidateIndex  = "candidate_index"
	progressDetailRawResponse     = "raw_response"
	progressDetailRawResponseSize = "raw_response_bytes"
)

func (runner *Runner) emitStepStarted(ctx context.Context, req *Request, step model.ProgressStep, capabilityID string, candidateIndex *int) {
	runner.emitStep(ctx, req, step, model.ProgressStatusStarted, "proposal "+string(step)+" started", capabilityID, candidateIndex, nil, nil)
}

func (runner *Runner) emitStepCompleted(ctx context.Context, req *Request, step model.ProgressStep, capabilityID string, candidateIndex *int, stage model.ProposalStage, attempt model.ProposalAttempt, err error) {
	status := proposalProgressStatus(ctx, err)
	message := "proposal " + string(step) + " " + string(status)
	runner.emitStep(ctx, req, step, status, message, capabilityID, candidateIndex, &stage, &attempt)
}

func (runner *Runner) emitStep(ctx context.Context, req *Request, step model.ProgressStep, status model.ProgressStatus, message string, capabilityID string, candidateIndex *int, stage *model.ProposalStage, attempt *model.ProposalAttempt) {
	now := time.Now().UTC()
	event := &model.ProgressEvent{
		ID:           model.ProgressID(now),
		Scope:        model.ProgressScopeAcquisition,
		Stage:        model.ProgressStageProposal,
		Step:         step,
		Status:       status,
		Message:      message,
		TraceID:      reqTraceID(req),
		ProviderID:   reqProviderID(req),
		CapabilityID: capabilityID,
		CreatedAt:    now.Format(time.RFC3339Nano),
		Details:      stepDetails(step, runner.modelName, candidateIndex, stage, attempt),
	}
	if attempt != nil && attempt.DurationMS > 0 {
		event.DurationMS = attempt.DurationMS
	}
	if attempt != nil && attempt.Error != nil {
		event.Error = attempt.Error
	}
	progress.Emit(ctx, event)
}

func stepDetails(step model.ProgressStep, modelName string, candidateIndex *int, stage *model.ProposalStage, attempt *model.ProposalAttempt) map[string]any {
	details := map[string]any{
		progressDetailProposalStage: string(step),
	}
	if modelName != "" {
		details[progressDetailModel] = modelName
	}
	if candidateIndex != nil {
		details[progressDetailCandidateIndex] = *candidateIndex
	}
	if stage != nil {
		for key, value := range stage.Summary {
			details[string(key)] = value
		}
	}
	if attempt != nil && attempt.RawResponse != "" {
		details[progressDetailRawResponse] = attempt.RawResponse
		details[progressDetailRawResponseSize] = len(attempt.RawResponse)
	}
	return details
}

func proposalProgressStatus(ctx context.Context, err error) model.ProgressStatus {
	if err == nil {
		return model.ProgressStatusSucceeded
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || (ctx != nil && ctx.Err() != nil) {
		return model.ProgressStatusCanceled
	}
	return model.ProgressStatusFailed
}

func reqTraceID(req *Request) string {
	if req == nil {
		return ""
	}
	return req.TraceID
}

func reqProviderID(req *Request) string {
	if req == nil || req.Provider == nil {
		return ""
	}
	return req.Provider.ID
}

func intPtr(value int) *int {
	return &value
}
