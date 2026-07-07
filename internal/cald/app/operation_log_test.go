package app

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestLogProgressWritesStructuredEvent(t *testing.T) {
	var buf bytes.Buffer
	restore := installTestLogger(&buf)
	defer restore()

	logProgress(context.Background(), &model.ProgressEvent{
		Scope:        model.ProgressScopeAcquisition,
		Stage:        model.ProgressStageProposal,
		Step:         model.ProgressStepProposalBinding,
		Status:       model.ProgressStatusFailed,
		Message:      "proposal binding failed",
		TraceID:      "trace_1",
		ProviderID:   "provider_1",
		CapabilityID: "document.convert",
		Details: map[string]any{
			"model":              "test-model",
			"selected":           0,
			"raw_response":       `{"candidates":[]}`,
			"raw_response_bytes": 17,
		},
		Error: &model.RecordError{Code: "proposal_failed", Message: "proposal failed"},
	})

	output := buf.String()
	for _, want := range []string{
		"event=progress",
		"scope=acquisition",
		"stage=proposal",
		"step=binding",
		"status=failed",
		"message=\"proposal binding failed\"",
		"trace_id=trace_1",
		"model=test-model",
		"selected=0",
		"raw_response_bytes=17",
		"error_code=proposal_failed",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("log output = %q, want %s", output, want)
		}
	}
	if strings.Contains(output, "candidates") || strings.Contains(output, "raw_response=") {
		t.Fatalf("log output = %q, want raw response omitted", output)
	}
}

func TestLogOperationDoesNotRequireProgressEvent(t *testing.T) {
	var buf bytes.Buffer
	restore := installTestLogger(&buf)
	defer restore()

	started := logOperationStart(context.Background(), opRun, slog.String("capability_id", "document.convert"))
	logOperationSuccess(context.Background(), opRun, started, slog.String("run_id", "run_1"))

	output := buf.String()
	for _, want := range []string{
		"event=operation_start",
		"event=operation_success",
		"op=run.create",
		"capability_id=document.convert",
		"run_id=run_1",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("log output = %q, want %s", output, want)
		}
	}
}

func installTestLogger(buf *bytes.Buffer) func() {
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	return func() {
		slog.SetDefault(previous)
	}
}
