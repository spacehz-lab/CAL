package execute

import (
	"context"
	"errors"
	"fmt"

	"github.com/spacehz-lab/cal/internal/model"
)

var (
	ErrMissingExecutor = errors.New("execute executor is not configured")
	ErrNilRequest      = errors.New("execute request is required")
	ErrNilExecution    = errors.New("execute execution is required")
	ErrUnsupportedKind = errors.New("unsupported execution kind")
)

// Runner dispatches execution requests to kind-specific executors.
type Runner struct {
	cli Executor
}

// NewRunner creates an execution runner.
func NewRunner(cli Executor) *Runner {
	return &Runner{cli: cli}
}

// Run dispatches a request by execution kind.
func (runner *Runner) Run(ctx context.Context, req *Request) (*Result, error) {
	if req == nil {
		return nil, ErrNilRequest
	}
	if req.Execution == nil {
		return nil, ErrNilExecution
	}
	switch req.Execution.Kind {
	case model.ExecutionKindCLI:
		if runner == nil || runner.cli == nil {
			return nil, ErrMissingExecutor
		}
		return runner.cli.Run(ctx, req)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedKind, req.Execution.Kind)
	}
}
