package probe

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/spacehz-lab/cal/internal/check"
	"github.com/spacehz-lab/cal/internal/execute"
	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/proposal"
)

func TestRunnerRunPassesExecuteVerification(t *testing.T) {
	exitCode := 0
	executor := &fakeExecutor{result: &execute.Result{Outputs: execute.Outputs{
		execute.OutputStdout:   {Kind: execute.OutputKindText, Text: "ok\n"},
		execute.OutputStderr:   {Kind: execute.OutputKindText, Text: ""},
		execute.OutputExitCode: {Kind: execute.OutputKindNumber, Number: &exitCode},
	}}}
	runner := NewRunner(execute.NewRunner(executor), check.NewChecker(), Options{KeepWorkdir: true})

	result, err := runner.Run(context.Background(), request(t, executeVerify(model.VerifySubjectStdout, "", model.VerifyPredicateNonEmpty)))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(result.Probes) != 1 || !result.Probes[0].Passed {
		t.Fatalf("probes = %#v, want one passed probe", result.Probes)
	}
	if result.Probes[0].Reason != ReasonVerifyChecksPassed || len(result.Probes[0].Evidence) != 1 {
		t.Fatalf("probe = %#v, want verify evidence", result.Probes[0])
	}
	if executor.called != 1 {
		t.Fatalf("executor called = %d, want 1", executor.called)
	}
}

func TestRunnerRunRecordsExecutionFailureAndContinues(t *testing.T) {
	executor := &fakeExecutor{err: errors.New("boom")}
	runner := NewRunner(execute.NewRunner(executor), check.NewChecker(), Options{})

	result, err := runner.Run(context.Background(), request(t, executeVerify(model.VerifySubjectStdout, "", model.VerifyPredicateNonEmpty)))
	if err != nil {
		t.Fatalf("Run() error = %v, want candidate failure to be recorded", err)
	}
	probe := result.Probes[0]
	if probe.Passed || probe.Reason != ReasonExecutionFailed || probe.Error.Code != CodeExecutionFailed {
		t.Fatalf("probe = %#v, want execution failure", probe)
	}
}

func TestRunnerRunClassifiesExecutionTimeout(t *testing.T) {
	executor := &fakeExecutor{waitForCancel: true}
	runner := NewRunner(execute.NewRunner(executor), check.NewChecker(), Options{Timeout: time.Millisecond})

	result, err := runner.Run(context.Background(), request(t, executeVerify(model.VerifySubjectStdout, "", model.VerifyPredicateNonEmpty)))
	if err != nil {
		t.Fatalf("Run() error = %v, want timeout probe", err)
	}
	probe := result.Probes[0]
	if probe.Passed || probe.Reason != ReasonExecutionTimeout || probe.Error.Code != CodeExecutionTimeout {
		t.Fatalf("probe = %#v, want execution timeout", probe)
	}
}

func TestRunnerRunRecordsVerificationFailure(t *testing.T) {
	executor := &fakeExecutor{result: &execute.Result{Outputs: execute.Outputs{
		execute.OutputStdout: {Kind: execute.OutputKindText, Text: ""},
	}}}
	runner := NewRunner(execute.NewRunner(executor), check.NewChecker(), Options{})

	result, err := runner.Run(context.Background(), request(t, executeVerify(model.VerifySubjectStdout, "", model.VerifyPredicateNonEmpty)))
	if err != nil {
		t.Fatalf("Run() error = %v, want verification failure probe", err)
	}
	probe := result.Probes[0]
	if probe.Passed || probe.Reason != ReasonVerificationFailed || probe.Error.Code != CodeVerificationFailed {
		t.Fatalf("probe = %#v, want verification failure", probe)
	}
}

func TestRunnerRunAcceptsContractWithoutExecution(t *testing.T) {
	executor := &fakeExecutor{err: errors.New("must not execute")}
	runner := NewRunner(execute.NewRunner(executor), check.NewChecker(), Options{})

	result, err := runner.Run(context.Background(), request(t, model.VerifySpec{Level: model.VerifyLevelL1, Method: model.VerifyMethodContract}))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	probe := result.Probes[0]
	if !probe.Passed || probe.Reason != ReasonContractEvidence || len(probe.Evidence) != 1 {
		t.Fatalf("probe = %#v, want contract evidence", probe)
	}
	if executor.called != 0 {
		t.Fatalf("executor called = %d, want 0", executor.called)
	}
}

func TestRunnerRunRecordsL0WithoutExecution(t *testing.T) {
	executor := &fakeExecutor{err: errors.New("must not execute")}
	runner := NewRunner(execute.NewRunner(executor), check.NewChecker(), Options{})

	result, err := runner.Run(context.Background(), request(t, model.VerifySpec{Level: model.VerifyLevelL0, Method: model.VerifyMethodExecute}))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	probe := result.Probes[0]
	if probe.Passed || probe.Reason != ReasonVerificationLevelL0 || probe.Error != nil {
		t.Fatalf("probe = %#v, want L0 non-passed probe without error", probe)
	}
	if executor.called != 0 {
		t.Fatalf("executor called = %d, want 0", executor.called)
	}
}

func TestRunnerRunRecordsMaterializeFailure(t *testing.T) {
	runner := NewRunner(execute.NewRunner(&fakeExecutor{}), check.NewChecker(), Options{})
	req := request(t, executeVerify(model.VerifySubjectStdout, "", model.VerifyPredicateNonEmpty))
	req.Plans[0].Inputs = map[string]any{"target": "{{missing}}/out.txt"}

	result, err := runner.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run() error = %v, want materialize failure probe", err)
	}
	probe := result.Probes[0]
	if probe.Passed || probe.Reason != ReasonProbeMaterializeFailed || probe.Error.Code != CodeProbeMaterializeFailed {
		t.Fatalf("probe = %#v, want materialize failure", probe)
	}
}

func TestRunnerRunRejectsInvalidRequestShape(t *testing.T) {
	runner := NewRunner(execute.NewRunner(&fakeExecutor{}), check.NewChecker(), Options{})
	req := request(t, executeVerify(model.VerifySubjectStdout, "", model.VerifyPredicateNonEmpty))
	req.Plans[0].CandidateIndex = 3

	_, err := runner.Run(context.Background(), req)
	if err == nil {
		t.Fatal("Run() error = nil, want invalid request error")
	}
	var probeErr *Error
	if !errors.As(err, &probeErr) || probeErr.Code != CodeInvalidProbeInput {
		t.Fatalf("Run() error = %#v, want CodeInvalidProbeInput", err)
	}
}

func request(t *testing.T, verify model.VerifySpec) *Request {
	t.Helper()
	return &Request{
		Provider: &model.Provider{ID: "provider_test", Kind: model.ProviderKindCLI, Path: "/bin/test"},
		Candidates: []model.Candidate{{
			ProviderID:   "provider_test",
			CapabilityID: "document.convert",
			Execution:    model.Execution{Kind: model.ExecutionKindCLI, Spec: map[string]any{model.ExecutionSpecArgs: []any{"convert", "{{source}}", "{{target}}"}}},
		}},
		Plans: []proposal.ProbePlan{{
			CandidateIndex: 0,
			Inputs:         map[string]any{"target": "{{workdir}}/out.txt"},
			Fixtures:       []proposal.Fixture{{Input: "source", Filename: "input.txt", Content: "hello\n"}},
			Verify:         verify,
		}},
		WorkRoot: filepath.Join(t.TempDir(), "trace"),
		Now:      func() time.Time { return time.Unix(100, 0) },
	}
}

func executeVerify(subject model.VerifySubjectType, input string, predicate model.VerifyPredicate) model.VerifySpec {
	return model.VerifySpec{
		Level:  model.VerifyLevelL1,
		Method: model.VerifyMethodExecute,
		Checks: []model.VerifyCheck{{Subject: model.VerifySubject{Type: subject, Input: input}, Predicate: predicate}},
	}
}

type fakeExecutor struct {
	called        int
	result        *execute.Result
	err           error
	waitForCancel bool
}

func (executor *fakeExecutor) Run(ctx context.Context, _ *execute.Request) (*execute.Result, error) {
	executor.called++
	if executor.waitForCancel {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return executor.result, executor.err
}
