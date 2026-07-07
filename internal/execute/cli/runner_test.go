package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/spacehz-lab/cal/internal/execute"
	"github.com/spacehz-lab/cal/internal/model"
)

func TestRunnerRejectsProviderKindMismatch(t *testing.T) {
	runner := NewRunner()
	_, err := runner.Run(context.Background(), &execute.Request{
		Provider:  &model.Provider{Kind: model.ProviderKindApp, Path: "/bin/echo"},
		Execution: &model.Execution{Kind: model.ExecutionKindCLI, Spec: map[string]any{model.ExecutionSpecArgs: []string{}}},
	})
	if !errors.Is(err, ErrProviderKind) {
		t.Fatalf("Run() error = %v, want ErrProviderKind", err)
	}
}

func TestRunnerRejectsExecutionKindMismatch(t *testing.T) {
	runner := NewRunner()
	_, err := runner.Run(context.Background(), &execute.Request{
		Provider:  &model.Provider{Kind: model.ProviderKindCLI, Path: "/bin/echo"},
		Execution: &model.Execution{Kind: model.ExecutionKindURLOpen, Spec: map[string]any{model.ExecutionSpecArgs: []string{}}},
	})
	if !errors.Is(err, ErrExecutionKind) {
		t.Fatalf("Run() error = %v, want ErrExecutionKind", err)
	}
}
