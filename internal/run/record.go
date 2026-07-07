package run

import (
	"fmt"
	"strings"
	"time"

	"github.com/spacehz-lab/cal/internal/model"
)

const (
	ErrorInvalidRunInput    = "invalid_run_input"
	ErrorCapabilityNotFound = "capability_not_found"
	ErrorBindingNotFound    = "binding_not_found"
	ErrorProviderNotFound   = "provider_not_found"
	ErrorExecutionFailed    = "execution_failed"
	ErrorVerificationFailed = "verification_failed"
	ErrorRunStoreFailed     = "run_store_failed"
)

func newRun(req *Request, now time.Time) *model.Run {
	started := now.UTC()
	return &model.Run{
		ID:           newRunID(started),
		CapabilityID: strings.TrimSpace(req.CapabilityID),
		Inputs:       req.Inputs,
		StartedAt:    started.Format(time.RFC3339Nano),
	}
}

func finishSucceeded(run *model.Run, started time.Time) {
	finished := time.Now().UTC()
	run.Status = model.RunStatusSucceeded
	run.FinishedAt = finished.Format(time.RFC3339Nano)
	run.DurationMS = finished.Sub(started.UTC()).Milliseconds()
	run.Error = nil
}

func finishFailed(run *model.Run, started time.Time, code string, message string) {
	finished := time.Now().UTC()
	run.Status = model.RunStatusFailed
	run.Verified = false
	run.FinishedAt = finished.Format(time.RFC3339Nano)
	run.DurationMS = finished.Sub(started.UTC()).Milliseconds()
	run.Error = &model.RecordError{Code: code, Message: message}
}

func newRunID(now time.Time) string {
	return "run_" + fmt.Sprint(now.UTC().UnixNano())
}
