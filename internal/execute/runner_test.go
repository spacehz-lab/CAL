package execute

import (
	"context"
	"errors"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestRunnerDispatchesCLIExecutor(t *testing.T) {
	executor := &fakeExecutor{result: &Result{Outputs: Outputs{OutputText: {Kind: OutputKindText, Text: "ok"}}}}
	runner := NewRunner(executor)
	req := &Request{Execution: &model.Execution{Kind: model.ExecutionKindCLI}}
	result, err := runner.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Outputs[OutputText].Text != "ok" {
		t.Fatalf("Run() result = %#v, want output text", result)
	}
	if executor.called != 1 {
		t.Fatalf("executor calls = %d, want 1", executor.called)
	}
}

func TestRunnerRejectsUnsupportedKind(t *testing.T) {
	runner := NewRunner(&fakeExecutor{})
	req := &Request{Execution: &model.Execution{Kind: model.ExecutionKindURLOpen}}
	_, err := runner.Run(context.Background(), req)
	if !errors.Is(err, ErrUnsupportedKind) {
		t.Fatalf("Run() error = %v, want ErrUnsupportedKind", err)
	}
}

func TestRunnerRejectsMissingCLIExecutor(t *testing.T) {
	runner := NewRunner(nil)
	req := &Request{Execution: &model.Execution{Kind: model.ExecutionKindCLI}}
	_, err := runner.Run(context.Background(), req)
	if !errors.Is(err, ErrMissingExecutor) {
		t.Fatalf("Run() error = %v, want ErrMissingExecutor", err)
	}
}

type fakeExecutor struct {
	called int
	result *Result
	err    error
}

func (executor *fakeExecutor) Run(context.Context, *Request) (*Result, error) {
	executor.called++
	return executor.result, executor.err
}
