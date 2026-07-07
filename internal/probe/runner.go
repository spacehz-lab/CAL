package probe

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spacehz-lab/cal/internal/check"
	"github.com/spacehz-lab/cal/internal/execute"
	"github.com/spacehz-lab/cal/internal/model"
)

// Runner verifies proposed candidate bindings before promotion.
type Runner struct {
	executor *execute.Runner
	checker  *check.Checker
	options  Options
}

// NewRunner creates an acquisition probe runner.
func NewRunner(executor *execute.Runner, checker *check.Checker, options Options) *Runner {
	return &Runner{
		executor: executor,
		checker:  checker,
		options:  normalizeOptions(options),
	}
}

// Run probes candidates serially and returns trace-ready probe records.
func (runner *Runner) Run(ctx context.Context, req *Request) (*Result, error) {
	if err := runner.validate(req); err != nil {
		return nil, err
	}
	targets, err := buildTargets(req)
	if err != nil {
		return nil, err
	}
	result := &Result{Probes: make([]model.Probe, 0, len(targets))}
	now := req.now()
	for index := range targets {
		record, err := runner.runTarget(ctx, req.Provider, &targets[index], now)
		result.Probes = append(result.Probes, record)
		if err != nil && isStructuralProbeError(err) {
			return result, err
		}
	}
	return result, nil
}

func (runner *Runner) runTarget(ctx context.Context, provider *model.Provider, target *Target, now time.Time) (model.Probe, error) {
	cleanup, err := prepareWorkDir(target.WorkDir, runner.options.KeepWorkdir)
	if err != nil {
		return materializeFailedProbe(target, target.Plan.Verify, err, now), err
	}
	defer cleanup()

	plan, err := Materialize(target)
	if err != nil {
		return materializeFailedProbe(target, target.Plan.Verify, err, now), err
	}
	if err := runner.checker.Validate(&plan.Verify); err != nil {
		return invalidVerifyProbe(target, plan, err, now), err
	}
	if plan.Verify.Level == model.VerifyLevelL0 {
		return l0Probe(target, plan, now), nil
	}
	switch plan.Verify.Method {
	case model.VerifyMethodContract:
		return contractProbe(target, plan, now), nil
	case model.VerifyMethodExecute:
		return runner.executeAndCheck(ctx, provider, target, plan, now)
	default:
		err := newError(CodeUnsupportedVerifyMethod, "verify method is not supported")
		return invalidVerifyProbe(target, plan, err, now), err
	}
}

func (runner *Runner) executeAndCheck(ctx context.Context, provider *model.Provider, target *Target, plan *MaterializedPlan, now time.Time) (model.Probe, error) {
	executeCtx, cancel := context.WithTimeout(ctx, runner.options.Timeout)
	defer cancel()
	executionResult, err := runner.executor.Run(executeCtx, &execute.Request{
		Provider:  provider,
		Execution: &target.Candidate.Execution,
		Inputs:    plan.Inputs,
	})
	if err != nil {
		if ctxErr := executeCtx.Err(); ctxErr != nil {
			err = ctxErr
		}
		return executionFailedProbe(target, plan, err, now), err
	}
	if executionResult == nil {
		err := fmt.Errorf("execute result is required")
		return executionFailedProbe(target, plan, err, now), err
	}
	checkResult, err := runner.checker.Run(ctx, checkRequest(&plan.Verify, plan.Inputs, executionResult.Outputs))
	if err != nil {
		return verificationFailedProbe(target, plan, err, now), err
	}
	if checkResult == nil {
		err := fmt.Errorf("check result is required")
		return verificationFailedProbe(target, plan, err, now), err
	}
	return passedProbe(target, plan, checkResult.Evidence, now), nil
}

func (runner *Runner) validate(req *Request) error {
	if runner == nil {
		return newError(CodeInvalidProbeInput, "probe runner is required")
	}
	if runner.executor == nil {
		return newError(CodeInvalidProbeInput, "probe executor is required")
	}
	if runner.checker == nil {
		return newError(CodeInvalidProbeInput, "probe checker is required")
	}
	if req == nil {
		return newError(CodeInvalidProbeInput, "probe request is required")
	}
	if req.Provider == nil || strings.TrimSpace(req.Provider.ID) == "" {
		return newError(CodeInvalidProbeInput, "provider is required")
	}
	if strings.TrimSpace(req.WorkRoot) == "" {
		return newError(CodeInvalidProbeInput, "work root is required")
	}
	if len(req.Candidates) != len(req.Plans) {
		return newError(CodeInvalidProbeInput, "candidates and probe plans must have the same length")
	}
	return nil
}

func isStructuralProbeError(err error) bool {
	var coded *Error
	if !errors.As(err, &coded) {
		return false
	}
	return coded.Code == CodeInvalidProbeInput
}

func checkRequest(spec *model.VerifySpec, inputs map[string]any, outputs execute.Outputs) *check.Request {
	return &check.Request{
		Spec:     spec,
		Inputs:   inputs,
		Stdout:   textOutput(outputs, execute.OutputStdout),
		Stderr:   textOutput(outputs, execute.OutputStderr),
		ExitCode: numberOutput(outputs, execute.OutputExitCode),
	}
}

func textOutput(outputs execute.Outputs, name execute.OutputName) string {
	output, ok := outputs[name]
	if !ok {
		return ""
	}
	return output.Text
}

func numberOutput(outputs execute.Outputs, name execute.OutputName) int {
	output, ok := outputs[name]
	if !ok || output.Number == nil {
		return 0
	}
	return *output.Number
}
