package tracelog

import (
	"context"
	"strings"
	"time"

	"github.com/spacehz-lab/cal/internal/model"
)

// Writer assembles and persists acquisition trace records.
type Writer struct {
	store Store
	now   func() time.Time
}

// NewWriter creates a trace writer.
func NewWriter(store Store, now func() time.Time) *Writer {
	return &Writer{store: store, now: now}
}

// Start writes the initial running trace.
func (writer *Writer) Start(ctx context.Context, req *Request) (*Result, error) {
	return writer.write(ctx, req, model.TraceStatusRunning)
}

// Complete writes the completed trace with all stage results.
func (writer *Writer) Complete(ctx context.Context, req *Request) (*Result, error) {
	return writer.write(ctx, req, model.TraceStatusCompleted)
}

// Fail writes the failed trace with any partial stage results.
func (writer *Writer) Fail(ctx context.Context, req *Request) (*Result, error) {
	return writer.write(ctx, req, model.TraceStatusFailed)
}

// Cancel writes the canceled trace with any partial stage results.
func (writer *Writer) Cancel(ctx context.Context, req *Request) (*Result, error) {
	return writer.write(ctx, req, model.TraceStatusCanceled)
}

func (writer *Writer) write(ctx context.Context, req *Request, status model.TraceStatus) (*Result, error) {
	if err := writer.validate(req, status); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	now := nowUTC(writer.now)
	trace := buildTrace(req, status, now)
	if err := writer.store.SaveTrace(&trace); err != nil {
		return nil, wrapError(CodeTraceStoreFailed, "save trace", err)
	}
	return &Result{Trace: trace}, nil
}

func (writer *Writer) validate(req *Request, status model.TraceStatus) error {
	if writer == nil {
		return newError(CodeInvalidTraceInput, "trace writer is required")
	}
	if writer.store == nil {
		return newError(CodeInvalidTraceInput, "trace store is required")
	}
	if req == nil {
		return newError(CodeInvalidTraceInput, "trace request is required")
	}
	if status != model.TraceStatusRunning && strings.TrimSpace(req.TraceID) == "" {
		return newError(CodeInvalidTraceInput, "trace id is required")
	}
	return nil
}

func buildTrace(req *Request, status model.TraceStatus, now time.Time) model.Trace {
	startedAt := strings.TrimSpace(req.StartedAt)
	if startedAt == "" {
		startedAt = now.Format(time.RFC3339Nano)
	}
	traceID := strings.TrimSpace(req.TraceID)
	if traceID == "" {
		traceID = model.TraceID(now)
	}
	trace := model.Trace{
		ID:           traceID,
		StartedAt:    startedAt,
		Status:       status,
		Hint:         req.Hint,
		ProviderIDs:  req.ProviderIDs,
		Observations: req.Observations,
		Proposal:     req.Proposal,
		Candidates:   req.Candidates,
		Probes:       req.Probes,
		Promotions:   req.Promotions,
		Error:        req.Error,
	}
	if status != model.TraceStatusRunning {
		trace.EndedAt = now.Format(time.RFC3339Nano)
	}
	return trace
}
