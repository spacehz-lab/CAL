package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spacehz-lab/cal/internal/execute"
	"github.com/spacehz-lab/cal/internal/model"
)

var (
	ErrNilProvider         = errors.New("cli execute provider is required")
	ErrMissingPath         = errors.New("cli provider path is required")
	ErrProviderKind        = errors.New("provider kind cannot run cli execution")
	ErrExecutionKind       = errors.New("execution kind cannot run cli execution")
	ErrMissingStdoutTarget = errors.New("stdout path input is required")
)

// Runner executes CLI providers.
type Runner struct{}

// NewRunner creates a CLI execution runner.
func NewRunner() *Runner {
	return &Runner{}
}

// Run executes one CLI request and returns typed outputs.
func (runner *Runner) Run(ctx context.Context, req *execute.Request) (*execute.Result, error) {
	if err := validateRequest(req); err != nil {
		return nil, err
	}
	return runCommand(ctx, req)
}

func validateRequest(req *execute.Request) error {
	if req == nil {
		return execute.ErrNilRequest
	}
	if req.Provider == nil {
		return ErrNilProvider
	}
	if req.Execution == nil {
		return execute.ErrNilExecution
	}
	if req.Provider.Kind != model.ProviderKindCLI {
		return fmt.Errorf("%w: %s", ErrProviderKind, req.Provider.Kind)
	}
	if req.Execution.Kind != model.ExecutionKindCLI {
		return fmt.Errorf("%w: %s", ErrExecutionKind, req.Execution.Kind)
	}
	if strings.TrimSpace(req.Provider.Path) == "" {
		return ErrMissingPath
	}
	return nil
}
