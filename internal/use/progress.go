package use

import (
	"context"
	"time"

	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/progress"
)

// ProgressFunc observes live use progress.
type ProgressFunc func(context.Context, *model.ProgressEvent)

// Option configures a use runner.
type Option func(*Runner)

// WithProgress installs a best-effort progress observer.
func WithProgress(fn ProgressFunc) Option {
	return func(runner *Runner) {
		runner.onProgress = fn
	}
}

func (runner *Runner) emitProgress(ctx context.Context, result *Result, stage model.ProgressStage, status model.ProgressStatus, started time.Time, err *model.RecordError) {
	if runner == nil {
		return
	}
	now := time.Now().UTC()
	event := &model.ProgressEvent{
		ID:           model.ProgressID(now),
		Scope:        model.ProgressScopeUse,
		Stage:        stage,
		Status:       status,
		UseID:        useID(result),
		ProviderID:   useProviderID(result),
		CapabilityID: useCapabilityID(result),
		BindingID:    useBindingID(result),
		RunID:        useRunID(result),
		Error:        err,
		CreatedAt:    now.Format(time.RFC3339Nano),
	}
	if status != model.ProgressStatusStarted {
		event.DurationMS = now.Sub(started.UTC()).Milliseconds()
	}
	progress.Emit(ctx, event, progress.Handler(runner.onProgress))
}

func useRecordError(code string, message string) *model.RecordError {
	return &model.RecordError{Code: code, Message: message}
}

func useID(result *Result) string {
	if result == nil {
		return ""
	}
	return result.ID
}

func useProviderID(result *Result) string {
	if result == nil {
		return ""
	}
	if result.Selection != nil && result.Selection.ProviderID != "" {
		return result.Selection.ProviderID
	}
	if result.Run != nil {
		return result.Run.ProviderID
	}
	return ""
}

func useCapabilityID(result *Result) string {
	if result == nil {
		return ""
	}
	if result.Selection != nil && result.Selection.CapabilityID != "" {
		return result.Selection.CapabilityID
	}
	if result.Run != nil {
		return result.Run.CapabilityID
	}
	return ""
}

func useBindingID(result *Result) string {
	if result == nil {
		return ""
	}
	if result.Selection != nil && result.Selection.BindingID != "" {
		return result.Selection.BindingID
	}
	if result.Run != nil {
		return result.Run.BindingID
	}
	return ""
}

func useRunID(result *Result) string {
	if result == nil || result.Run == nil {
		return ""
	}
	return result.Run.ID
}
