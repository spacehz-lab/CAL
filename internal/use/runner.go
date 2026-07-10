package use

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spacehz-lab/cal/internal/llm"
	"github.com/spacehz-lab/cal/internal/model"
	runpkg "github.com/spacehz-lab/cal/internal/run"
	"github.com/spacehz-lab/cal/internal/use/plan"
	selector "github.com/spacehz-lab/cal/internal/use/select"
)

var (
	ErrMissingStore    = errors.New("use store is not configured")
	ErrMissingSelector = errors.New("use selector is not configured")
	ErrMissingPlanner  = errors.New("use planner is not configured")
	ErrMissingRunner   = errors.New("use run runner is not configured")
	ErrNilRequest      = errors.New("use request is required")
)

// Runner owns intent-level reuse orchestration.
type Runner struct {
	store    Store
	selector *selector.Runner
	planner  *plan.Runner
	executor Executor

	onProgress ProgressFunc
}

// NewRunner creates a use runner.
func NewRunner(store Store, selector *selector.Runner, planner *plan.Runner, executor Executor, opts ...Option) *Runner {
	runner := &Runner{
		store:    store,
		selector: selector,
		planner:  planner,
		executor: executor,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(runner)
		}
	}
	return runner
}

// NewDefaultRunner creates a use runner with the standard selector and planner.
func NewDefaultRunner(store Store, executor Executor, client llm.Client, opts ...Option) *Runner {
	return NewRunner(store, newSelector(client), plan.NewRunner(), executor, opts...)
}

// Run selects, plans, and executes one intent-level reuse request.
func (runner *Runner) Run(ctx context.Context, req *Request) (*Result, error) {
	if err := runner.validate(req); err != nil {
		return nil, err
	}
	normalized := normalizeRequest(req)
	started := time.Now().UTC()
	result := newResult(normalized.Intent, started)

	stageStarted := time.Now().UTC()
	runner.emitProgress(ctx, result, model.ProgressStageSelect, model.ProgressStatusStarted, stageStarted, nil)
	capabilities, err := runner.store.ListCapabilities()
	if err != nil {
		runner.emitProgress(ctx, result, model.ProgressStageSelect, model.ProgressStatusFailed, stageStarted, useRecordError(CodeUseStoreFailed, err.Error()))
		return nil, err
	}
	providers, err := runner.store.ListProviders()
	if err != nil {
		runner.emitProgress(ctx, result, model.ProgressStageSelect, model.ProgressStatusFailed, stageStarted, useRecordError(CodeUseStoreFailed, err.Error()))
		return nil, err
	}
	selection, err := runner.selector.Run(ctx, &selector.Request{
		Intent:           normalized.Intent,
		Inputs:           normalized.Inputs,
		ProviderID:       normalized.ProviderID,
		ForceLLM:         normalized.ForceLLMSelection,
		MinVerifyLevel:   normalized.MinVerifyLevel,
		Capabilities:     capabilities,
		ProviderCommands: providerCommands(providers),
	})
	if err != nil {
		code, message := selectionError(err)
		finishFailed(result, started, code, message)
		runner.emitProgress(ctx, result, model.ProgressStageSelect, model.ProgressStatusFailed, stageStarted, useRecordError(code, message))
		return result, nil
	}
	result.Selection = selectionResult(selection)
	runner.emitProgress(ctx, result, model.ProgressStageSelect, model.ProgressStatusSucceeded, stageStarted, nil)

	stageStarted = time.Now().UTC()
	runner.emitProgress(ctx, result, model.ProgressStagePlan, model.ProgressStatusStarted, stageStarted, nil)
	planned, err := runner.planner.Run(&plan.Request{
		UseID:          result.ID,
		Now:            started,
		Inputs:         normalized.Inputs,
		RequiredInputs: selection.RequiredInputs,
		InputsPatch:    selection.InputsPatch,
	})
	if err != nil {
		code, message := planError(err)
		finishFailed(result, started, code, message)
		runner.emitProgress(ctx, result, model.ProgressStagePlan, model.ProgressStatusFailed, stageStarted, useRecordError(code, message))
		return result, nil
	}
	runner.emitProgress(ctx, result, model.ProgressStagePlan, model.ProgressStatusSucceeded, stageStarted, nil)

	stageStarted = time.Now().UTC()
	runner.emitProgress(ctx, result, model.ProgressStageRun, model.ProgressStatusStarted, stageStarted, nil)
	runResult, err := runner.executor.Run(ctx, &runpkg.Request{
		CapabilityID:   selection.CapabilityID,
		BindingID:      selection.BindingID,
		ProviderID:     selection.ProviderID,
		Inputs:         planned.Inputs,
		Verify:         normalized.Verify,
		MinVerifyLevel: normalized.MinVerifyLevel,
	})
	if err != nil {
		runner.emitProgress(ctx, result, model.ProgressStageRun, model.ProgressStatusFailed, stageStarted, useRecordError(CodeRunFailed, err.Error()))
		return nil, err
	}
	if runResult == nil || runResult.Run == nil {
		finishFailed(result, started, CodeRunFailed, "run result is required")
		runner.emitProgress(ctx, result, model.ProgressStageRun, model.ProgressStatusFailed, stageStarted, useRecordError(CodeRunFailed, "run result is required"))
		return result, nil
	}
	result.Run = runResult.Run
	if runResult.Run.Status == model.RunStatusFailed {
		code := CodeRunFailed
		message := "selected binding run failed"
		if runResult.Run.Error != nil {
			code = runResult.Run.Error.Code
			message = runResult.Run.Error.Message
		}
		finishFailed(result, started, code, message)
		runner.emitProgress(ctx, result, model.ProgressStageRun, model.ProgressStatusFailed, stageStarted, useRecordError(code, message))
		return result, nil
	}
	runner.emitProgress(ctx, result, model.ProgressStageRun, model.ProgressStatusSucceeded, stageStarted, nil)
	finishSucceeded(result, started)
	return result, nil
}

func (runner *Runner) validate(req *Request) error {
	if runner == nil || runner.store == nil {
		return ErrMissingStore
	}
	if runner.selector == nil {
		return ErrMissingSelector
	}
	if runner.planner == nil {
		return ErrMissingPlanner
	}
	if runner.executor == nil {
		return ErrMissingRunner
	}
	if req == nil {
		return ErrNilRequest
	}
	if strings.TrimSpace(req.Intent) == "" {
		return fmt.Errorf("%s: intent is required", CodeInvalidUseInput)
	}
	return nil
}

func normalizeRequest(req *Request) *Request {
	normalized := *req
	normalized.Intent = strings.TrimSpace(req.Intent)
	if normalized.Inputs == nil {
		normalized.Inputs = map[string]any{}
	}
	if normalized.MinVerifyLevel == "" {
		normalized.MinVerifyLevel = model.VerifyLevelL2
	}
	return &normalized
}

func selectionResult(result *selector.Result) *Selection {
	return &Selection{
		Source:               string(result.Source),
		CapabilityID:         result.CapabilityID,
		BindingID:            result.BindingID,
		ProviderID:           result.ProviderID,
		Reason:               result.Reason,
		CandidatesConsidered: result.CandidatesConsidered,
	}
}

func providerCommands(providers []model.Provider) map[string]string {
	commands := make(map[string]string, len(providers))
	for _, provider := range providers {
		command := strings.TrimSpace(provider.Name)
		if command == "" {
			command = filepath.Base(provider.Path)
		}
		if provider.ID != "" && command != "" {
			commands[provider.ID] = command
		}
	}
	return commands
}

func selectionError(err error) (string, string) {
	var selectionErr *selector.Error
	if errors.As(err, &selectionErr) {
		return selectionErr.Code, selectionErr.Message
	}
	return CodeNoMatch, err.Error()
}

func planError(err error) (string, string) {
	switch {
	case errors.Is(err, plan.ErrMissingInputs):
		return CodeMissingInputs, err.Error()
	case errors.Is(err, plan.ErrInvalidInputsPatch):
		return CodeInvalidLLMSelection, err.Error()
	case errors.Is(err, plan.ErrArtifactPathFailed):
		return CodeArtifactPathFailed, err.Error()
	default:
		return CodeMissingInputs, err.Error()
	}
}

func newSelector(client llm.Client) *selector.Runner {
	if client == nil {
		return selector.NewRunner()
	}
	return selector.NewRunner(selector.WithLLM(client))
}
