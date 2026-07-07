package run

import (
	"context"
	"time"

	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/progress"
)

// ProgressFunc observes live run progress.
type ProgressFunc func(context.Context, *model.ProgressEvent)

// Option configures a run runner.
type Option func(*Runner)

// WithProgress installs a best-effort progress observer.
func WithProgress(fn ProgressFunc) Option {
	return func(runner *Runner) {
		runner.onProgress = fn
	}
}

func (runner *Runner) emitProgress(ctx context.Context, req *Request, run *model.Run, stage model.ProgressStage, status model.ProgressStatus, started time.Time, err *model.RecordError) {
	if runner == nil {
		return
	}
	now := time.Now().UTC()
	event := &model.ProgressEvent{
		ID:           model.ProgressID(now),
		Scope:        model.ProgressScopeRun,
		Stage:        stage,
		Status:       status,
		RunID:        runID(run),
		ProviderID:   runProviderID(run, req),
		CapabilityID: runCapabilityID(run, req),
		BindingID:    runBindingID(run, req),
		Error:        err,
		CreatedAt:    now.Format(time.RFC3339Nano),
	}
	if status != model.ProgressStatusStarted {
		event.DurationMS = now.Sub(started.UTC()).Milliseconds()
	}
	progress.Emit(ctx, event, progress.Handler(runner.onProgress))
}

func runRecordError(code string, message string) *model.RecordError {
	return &model.RecordError{Code: code, Message: message}
}

func runID(run *model.Run) string {
	if run == nil {
		return ""
	}
	return run.ID
}

func runProviderID(run *model.Run, req *Request) string {
	if run != nil && run.ProviderID != "" {
		return run.ProviderID
	}
	if req == nil {
		return ""
	}
	return req.ProviderID
}

func runCapabilityID(run *model.Run, req *Request) string {
	if run != nil && run.CapabilityID != "" {
		return run.CapabilityID
	}
	if req == nil {
		return ""
	}
	return req.CapabilityID
}

func runBindingID(run *model.Run, req *Request) string {
	if run != nil && run.BindingID != "" {
		return run.BindingID
	}
	if req == nil {
		return ""
	}
	return req.BindingID
}
