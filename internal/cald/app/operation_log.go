package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/spacehz-lab/cal/internal/model"
)

const (
	logEventOperationStart   = "operation_start"
	logEventOperationSuccess = "operation_success"
	logEventOperationFailure = "operation_failure"
	logEventProgress         = "progress"
	logDetailRawResponse     = "raw_response"

	opAcquire = "acquisition.run"
	opRun     = "run.create"
	opUse     = "use.run"
)

func logOperationStart(ctx context.Context, op string, attrs ...slog.Attr) time.Time {
	started := time.Now().UTC()
	values := []any{
		slog.String("event", logEventOperationStart),
		slog.String("op", op),
	}
	values = append(values, slogAttrs(attrs)...)
	slog.InfoContext(ctx, "cal operation started", values...)
	return started
}

func logOperationSuccess(ctx context.Context, op string, started time.Time, attrs ...slog.Attr) {
	values := []any{
		slog.String("event", logEventOperationSuccess),
		slog.String("op", op),
		slog.Int64("duration_ms", time.Since(started.UTC()).Milliseconds()),
	}
	values = append(values, slogAttrs(attrs)...)
	slog.InfoContext(ctx, "cal operation succeeded", values...)
}

func logOperationFailure(ctx context.Context, op string, started time.Time, err error, attrs ...slog.Attr) {
	values := []any{
		slog.String("event", logEventOperationFailure),
		slog.String("op", op),
		slog.Int64("duration_ms", time.Since(started.UTC()).Milliseconds()),
	}
	values = append(values, slogAttrs(attrs)...)
	if err != nil {
		values = append(values, slog.String("error", err.Error()))
	}
	slog.ErrorContext(ctx, "cal operation failed", values...)
}

func logProgress(ctx context.Context, event *model.ProgressEvent) {
	if event == nil {
		return
	}
	values := []any{
		slog.String("event", logEventProgress),
		slog.String("scope", string(event.Scope)),
		slog.String("stage", string(event.Stage)),
		slog.String("status", string(event.Status)),
	}
	values = appendString(values, "step", string(event.Step))
	values = appendString(values, "message", event.Message)
	values = appendString(values, "trace_id", event.TraceID)
	values = appendString(values, "run_id", event.RunID)
	values = appendString(values, "use_id", event.UseID)
	values = appendString(values, "provider_id", event.ProviderID)
	values = appendString(values, "capability_id", event.CapabilityID)
	values = appendString(values, "binding_id", event.BindingID)
	values = appendSafeDetails(values, event.Details)
	if event.DurationMS > 0 {
		values = append(values, slog.Int64("duration_ms", event.DurationMS))
	}
	if event.Error != nil {
		values = appendString(values, "error_code", event.Error.Code)
		values = appendString(values, "error", event.Error.Message)
	}
	if event.Status == model.ProgressStatusFailed {
		slog.ErrorContext(ctx, "cal progress failed", values...)
		return
	}
	slog.InfoContext(ctx, "cal progress", values...)
}

func slogAttrs(attrs []slog.Attr) []any {
	values := make([]any, 0, len(attrs))
	for _, attr := range attrs {
		values = append(values, attr)
	}
	return values
}

func appendString(values []any, key string, value string) []any {
	if value == "" {
		return values
	}
	return append(values, slog.String(key, value))
}

func appendSafeDetails(values []any, details map[string]any) []any {
	for key, value := range details {
		if key == "" || key == logDetailRawResponse {
			continue
		}
		switch typed := value.(type) {
		case string:
			values = appendString(values, key, typed)
		case int:
			values = append(values, slog.Int(key, typed))
		case int64:
			values = append(values, slog.Int64(key, typed))
		case float64:
			values = append(values, slog.Float64(key, typed))
		case bool:
			values = append(values, slog.Bool(key, typed))
		}
	}
	return values
}
