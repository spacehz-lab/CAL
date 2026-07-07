package run

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spacehz-lab/cal/internal/execute"
	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/run/resolve"
)

var (
	ErrMissingStore    = errors.New("run store is not configured")
	ErrMissingResolver = errors.New("run resolver is not configured")
	ErrMissingExecutor = errors.New("run executor is not configured")
	ErrMissingChecker  = errors.New("run checker is not configured")
	ErrNilRequest      = errors.New("run request is required")
)

// Runner executes promoted capabilities and records durable runs.
type Runner struct {
	store    Store
	resolver *resolve.Runner
	executor Executor
	checker  Checker

	onProgress ProgressFunc
}

// NewRunner creates a formal run runner.
func NewRunner(store Store, resolver *resolve.Runner, executor Executor, checker Checker, opts ...Option) *Runner {
	runner := &Runner{
		store:    store,
		resolver: resolver,
		executor: executor,
		checker:  checker,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(runner)
		}
	}
	return runner
}

// NewDefaultRunner creates a run runner with the standard resolver.
func NewDefaultRunner(store Store, executor Executor, checker Checker, opts ...Option) *Runner {
	return NewRunner(store, resolve.NewRunner(), executor, checker, opts...)
}

// Run executes one promoted capability run.
func (runner *Runner) Run(ctx context.Context, req *Request) (*Result, error) {
	if err := runner.validate(req); err != nil {
		return nil, err
	}
	started := time.Now().UTC()
	record := newRun(req, started)

	stageStarted := time.Now().UTC()
	runner.emitProgress(ctx, req, record, model.ProgressStageResolve, model.ProgressStatusStarted, stageStarted, nil)
	capability, ok, err := runner.store.GetCapability(strings.TrimSpace(req.CapabilityID))
	if err != nil {
		runner.emitProgress(ctx, req, record, model.ProgressStageResolve, model.ProgressStatusFailed, stageStarted, runRecordError(ErrorRunStoreFailed, err.Error()))
		return nil, err
	}
	if !ok {
		message := fmt.Sprintf("capability %q was not found", req.CapabilityID)
		runner.emitProgress(ctx, req, record, model.ProgressStageResolve, model.ProgressStatusFailed, stageStarted, runRecordError(ErrorCapabilityNotFound, message))
		return runner.fail(ctx, record, started, ErrorCapabilityNotFound, message)
	}

	resolution, err := runner.resolver.Run(&resolve.Request{
		Capability:     &capability,
		BindingID:      req.BindingID,
		ProviderID:     req.ProviderID,
		Inputs:         req.Inputs,
		MinVerifyLevel: req.MinVerifyLevel,
	})
	if err != nil {
		runner.emitProgress(ctx, req, record, model.ProgressStageResolve, model.ProgressStatusFailed, stageStarted, runRecordError(ErrorBindingNotFound, err.Error()))
		return runner.fail(ctx, record, started, ErrorBindingNotFound, err.Error())
	}
	record.BindingID = resolution.Binding.ID
	record.ProviderID = resolution.Binding.ProviderID

	provider, ok, err := runner.store.GetProvider(resolution.Binding.ProviderID)
	if err != nil {
		runner.emitProgress(ctx, req, record, model.ProgressStageResolve, model.ProgressStatusFailed, stageStarted, runRecordError(ErrorRunStoreFailed, err.Error()))
		return nil, err
	}
	if !ok {
		message := fmt.Sprintf("provider %q was not found", resolution.Binding.ProviderID)
		runner.emitProgress(ctx, req, record, model.ProgressStageResolve, model.ProgressStatusFailed, stageStarted, runRecordError(ErrorProviderNotFound, message))
		return runner.fail(ctx, record, started, ErrorProviderNotFound, message)
	}
	if missing := missingInputs(resolution.RequiredInputs, req.Inputs); len(missing) > 0 {
		message := fmt.Sprintf("missing required input: %s", strings.Join(missing, ", "))
		runner.emitProgress(ctx, req, record, model.ProgressStageResolve, model.ProgressStatusFailed, stageStarted, runRecordError(ErrorInvalidRunInput, message))
		return runner.fail(ctx, record, started, ErrorInvalidRunInput, message)
	}
	runner.emitProgress(ctx, req, record, model.ProgressStageResolve, model.ProgressStatusSucceeded, stageStarted, nil)

	stageStarted = time.Now().UTC()
	runner.emitProgress(ctx, req, record, model.ProgressStageExecute, model.ProgressStatusStarted, stageStarted, nil)
	executionResult, err := runner.executor.Run(ctx, &execute.Request{
		Provider:  &provider,
		Execution: &resolution.Binding.Execution,
		Inputs:    req.Inputs,
	})
	if err != nil {
		runner.emitProgress(ctx, req, record, model.ProgressStageExecute, model.ProgressStatusFailed, stageStarted, runRecordError(ErrorExecutionFailed, err.Error()))
		return runner.fail(ctx, record, started, ErrorExecutionFailed, err.Error())
	}
	if executionResult == nil {
		message := "execute result is required"
		runner.emitProgress(ctx, req, record, model.ProgressStageExecute, model.ProgressStatusFailed, stageStarted, runRecordError(ErrorExecutionFailed, message))
		return runner.fail(ctx, record, started, ErrorExecutionFailed, message)
	}
	record.Outputs = durableOutputs(executionResult.Outputs)
	runner.emitProgress(ctx, req, record, model.ProgressStageExecute, model.ProgressStatusSucceeded, stageStarted, nil)

	if req.Verify {
		stageStarted = time.Now().UTC()
		runner.emitProgress(ctx, req, record, model.ProgressStageVerify, model.ProgressStatusStarted, stageStarted, nil)
		if resolution.Binding.Verify == nil {
			message := "selected binding does not declare verify spec"
			runner.emitProgress(ctx, req, record, model.ProgressStageVerify, model.ProgressStatusFailed, stageStarted, runRecordError(ErrorVerificationFailed, message))
			return runner.fail(ctx, record, started, ErrorVerificationFailed, message)
		}
		checkResult, err := runner.checker.Run(ctx, checkRequest(resolution.Binding.Verify, req.Inputs, executionResult.Outputs))
		if err != nil {
			runner.emitProgress(ctx, req, record, model.ProgressStageVerify, model.ProgressStatusFailed, stageStarted, runRecordError(ErrorVerificationFailed, err.Error()))
			return runner.fail(ctx, record, started, ErrorVerificationFailed, err.Error())
		}
		if checkResult == nil {
			message := "check result is required"
			runner.emitProgress(ctx, req, record, model.ProgressStageVerify, model.ProgressStatusFailed, stageStarted, runRecordError(ErrorVerificationFailed, message))
			return runner.fail(ctx, record, started, ErrorVerificationFailed, message)
		}
		record.Evidence = checkResult.Evidence
		record.Outputs = mergeOutputs(executionResult.Outputs, checkResult.Outputs)
		record.Verified = true
		runner.emitProgress(ctx, req, record, model.ProgressStageVerify, model.ProgressStatusSucceeded, stageStarted, nil)
		finishSucceeded(record, started)
		return runner.save(ctx, record, executionResult.Outputs, checkResult.Evidence)
	}
	if nonZeroExit(executionResult.Outputs) {
		message := "execution exited with non-zero status"
		return runner.fail(ctx, record, started, ErrorExecutionFailed, message)
	}
	finishSucceeded(record, started)
	return runner.save(ctx, record, executionResult.Outputs, nil)
}

func (runner *Runner) validate(req *Request) error {
	if runner == nil {
		return ErrMissingStore
	}
	if runner.store == nil {
		return ErrMissingStore
	}
	if runner.resolver == nil {
		return ErrMissingResolver
	}
	if runner.executor == nil {
		return ErrMissingExecutor
	}
	if req == nil {
		return ErrNilRequest
	}
	if strings.TrimSpace(req.CapabilityID) == "" {
		return fmt.Errorf("%s: capability_id is required", ErrorInvalidRunInput)
	}
	if req.Inputs == nil {
		return fmt.Errorf("%s: inputs must be an object", ErrorInvalidRunInput)
	}
	if req.Verify && runner.checker == nil {
		return ErrMissingChecker
	}
	return nil
}

func (runner *Runner) fail(ctx context.Context, record *model.Run, started time.Time, code string, message string) (*Result, error) {
	finishFailed(record, started, code, message)
	stageStarted := time.Now().UTC()
	runner.emitProgress(ctx, nil, record, model.ProgressStageRecord, model.ProgressStatusStarted, stageStarted, nil)
	if err := runner.store.SaveRun(record); err != nil {
		runner.emitProgress(ctx, nil, record, model.ProgressStageRecord, model.ProgressStatusFailed, stageStarted, runRecordError(ErrorRunStoreFailed, err.Error()))
		return nil, fmt.Errorf("%s: %w", ErrorRunStoreFailed, err)
	}
	runner.emitProgress(ctx, nil, record, model.ProgressStageRecord, model.ProgressStatusSucceeded, stageStarted, nil)
	return &Result{Run: record}, nil
}

func (runner *Runner) save(ctx context.Context, record *model.Run, outputs execute.Outputs, evidence []model.EvidenceRef) (*Result, error) {
	stageStarted := time.Now().UTC()
	runner.emitProgress(ctx, nil, record, model.ProgressStageRecord, model.ProgressStatusStarted, stageStarted, nil)
	if err := runner.store.SaveRun(record); err != nil {
		runner.emitProgress(ctx, nil, record, model.ProgressStageRecord, model.ProgressStatusFailed, stageStarted, runRecordError(ErrorRunStoreFailed, err.Error()))
		return nil, fmt.Errorf("%s: %w", ErrorRunStoreFailed, err)
	}
	runner.emitProgress(ctx, nil, record, model.ProgressStageRecord, model.ProgressStatusSucceeded, stageStarted, nil)
	return &Result{Run: record, Outputs: outputs, Evidence: evidence}, nil
}

func missingInputs(required []string, inputs map[string]any) []string {
	var missing []string
	for _, name := range required {
		value, ok := inputs[name]
		if !ok || value == nil || value == "" {
			missing = append(missing, name)
		}
	}
	return missing
}
