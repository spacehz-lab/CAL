package use

import (
	"fmt"
	"strings"
	"time"

	"github.com/spacehz-lab/cal/internal/model"
)

func newResult(intent string, now time.Time) *Result {
	started := now.UTC()
	return &Result{
		ID:        newUseID(started),
		Intent:    strings.TrimSpace(intent),
		StartedAt: started.Format(time.RFC3339Nano),
	}
}

func finishSucceeded(result *Result, started time.Time) {
	finished := time.Now().UTC()
	result.Status = model.RunStatusSucceeded
	result.FinishedAt = finished.Format(time.RFC3339Nano)
	result.DurationMS = finished.Sub(started.UTC()).Milliseconds()
	result.Error = nil
}

func finishFailed(result *Result, started time.Time, code string, message string) {
	finished := time.Now().UTC()
	result.Status = model.RunStatusFailed
	result.FinishedAt = finished.Format(time.RFC3339Nano)
	result.DurationMS = finished.Sub(started.UTC()).Milliseconds()
	result.Error = &model.RecordError{Code: code, Message: message}
}

func newUseID(now time.Time) string {
	return "use_" + fmt.Sprint(now.UTC().UnixNano())
}
