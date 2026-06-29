package control

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/runtime"
)

// RunRequest describes one service-side capability execution.
type RunRequest struct {
	CapabilityID   string           `json:"capability_id"`
	BindingID      string           `json:"binding_id,omitempty"`
	Inputs         map[string]any   `json:"inputs"`
	ProviderID     string           `json:"provider_id,omitempty"`
	Strategy       string           `json:"strategy,omitempty"`
	Verify         bool             `json:"verify,omitempty"`
	MinVerifyLevel core.VerifyLevel `json:"min_verify_level,omitempty"`
}

// Run executes a promoted capability binding and records the run.
func (svc Service) Run(ctx context.Context, req RunRequest) (core.Run, error) {
	if strings.TrimSpace(req.CapabilityID) == "" {
		return core.Run{}, NewAPIError("invalid_run_input", "capability_id is required")
	}
	if req.Inputs == nil {
		return core.Run{}, NewAPIError("invalid_run_input", "inputs must be an object")
	}
	started := time.Now().UTC()
	run := core.Run{
		ID:           newRunID(started),
		CapabilityID: req.CapabilityID,
		Inputs:       req.Inputs,
		StartedAt:    started.Format(time.RFC3339Nano),
	}

	runner := runtime.NewRunner(runtime.DefaultRegistry())
	capability, ok, err := svc.store.GetCapability(req.CapabilityID)
	if err != nil {
		return core.Run{}, err
	}
	if !ok {
		return svc.failRun(run, started, "capability_not_found", fmt.Sprintf("capability %q was not found", req.CapabilityID))
	}
	resolution, err := runner.Resolve(capability, runtime.ResolveOptions{BindingID: req.BindingID, ProviderID: req.ProviderID, Strategy: req.Strategy, MinVerifyLevel: req.MinVerifyLevel})
	if err != nil {
		return svc.failRun(run, started, "binding_not_found", err.Error())
	}
	run.BindingID = resolution.Binding.ID
	run.ProviderID = resolution.Binding.ProviderID

	provider, ok, err := svc.store.GetProvider(resolution.Binding.ProviderID)
	if err != nil {
		return core.Run{}, err
	}
	if !ok {
		return svc.failRun(run, started, "provider_not_found", fmt.Sprintf("provider %q was not found", resolution.Binding.ProviderID))
	}
	if err := runner.Validate(resolution.Binding, req.Inputs); err != nil {
		return svc.failRun(run, started, "invalid_run_input", err.Error())
	}
	executionResult, err := runner.Execute(ctx, provider, resolution.Binding.Execution, req.Inputs)
	if err != nil {
		return svc.failRun(run, started, "execution_failed", err.Error())
	}

	run.Status = core.RunStatusSucceeded
	if req.Verify {
		if resolution.Binding.Verify == nil {
			return svc.failRun(run, started, "verification_failed", "selected binding does not declare verify spec")
		}
		evidence, outputs, err := runner.Verify(ctx, *resolution.Binding.Verify, req.Inputs, executionResult)
		if err != nil {
			return svc.failRun(run, started, "verification_failed", err.Error())
		}
		run.Evidence = evidence
		run.Outputs = outputs
		run.Verified = true
	}
	finishRunSucceeded(&run, started)
	return run, svc.store.PutRun(run)
}

// GetRun returns one stored Run record.
func (svc Service) GetRun(id string) (core.Run, bool, error) {
	return svc.store.GetRun(id)
}

func (svc Service) failRun(run core.Run, started time.Time, code, message string) (core.Run, error) {
	finishRunFailed(&run, started, code, message)
	return run, svc.store.PutRun(run)
}

func finishRunFailed(run *core.Run, started time.Time, code, message string) {
	finished := time.Now().UTC()
	run.Status = core.RunStatusFailed
	run.Verified = false
	run.FinishedAt = finished.Format(time.RFC3339Nano)
	run.DurationMS = finished.Sub(started).Milliseconds()
	run.Error = &core.RecordError{Code: code, Message: message}
}

func finishRunSucceeded(run *core.Run, started time.Time) {
	finished := time.Now().UTC()
	run.Status = core.RunStatusSucceeded
	run.FinishedAt = finished.Format(time.RFC3339Nano)
	run.DurationMS = finished.Sub(started).Milliseconds()
}

func newRunID(now time.Time) string {
	return "run_" + fmt.Sprint(now.UTC().UnixNano())
}
