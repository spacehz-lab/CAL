package acquisition

import (
	"context"
	"time"

	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/progress"
)

// ProgressFunc observes live acquisition progress.
type ProgressFunc func(context.Context, *model.ProgressEvent)

// Option configures an acquisition runner.
type Option func(*Runner)

// WithProgress installs a best-effort progress observer.
func WithProgress(fn ProgressFunc) Option {
	return func(runner *Runner) {
		runner.onProgress = fn
	}
}

func (runner *Runner) emitProgress(ctx context.Context, req *Request, st *state, stage model.ProgressStage, status model.ProgressStatus, started time.Time, err *model.RecordError) {
	if runner == nil {
		return
	}
	now := time.Now().UTC()
	event := &model.ProgressEvent{
		ID:         model.ProgressID(now),
		Scope:      model.ProgressScopeAcquisition,
		Stage:      stage,
		Status:     status,
		TraceID:    st.traceID(),
		ProviderID: st.providerID(req),
		Error:      err,
		CreatedAt:  now.Format(time.RFC3339Nano),
	}
	if status != model.ProgressStatusStarted {
		event.DurationMS = now.Sub(started.UTC()).Milliseconds()
	}
	progress.Emit(ctx, event, progress.Handler(runner.onProgress))
}

func progressStatus(ctx context.Context, err error) model.ProgressStatus {
	if isContextDone(ctx, err) {
		return model.ProgressStatusCanceled
	}
	if err != nil {
		return model.ProgressStatusFailed
	}
	return model.ProgressStatusSucceeded
}
