package replay

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spacehz-lab/cal/internal/proposal"
)

// Runner turns a replay proposal file into normal proposal output.
type Runner struct {
	path string
}

// NewRunner creates a replay proposal runner.
func NewRunner(path string) *Runner {
	return &Runner{path: path}
}

// Run reads a replay proposal and returns normalized candidates and probe plans.
func (runner *Runner) Run(ctx context.Context, req *proposal.Request) (*proposal.Result, error) {
	if err := runner.validate(req); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	content, err := os.ReadFile(strings.TrimSpace(runner.path))
	if err != nil {
		return nil, fmt.Errorf("read replay proposal: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	parsed, err := parse(content, req.Provider.ID)
	if err != nil {
		return nil, err
	}
	normalized, err := proposal.NormalizeResult(parsed, proposal.NormalizeOptions{
		ProviderID: req.Provider.ID,
	})
	if err != nil {
		return nil, err
	}
	if len(normalized.Candidates) == 0 {
		return normalized, fmt.Errorf("replay proposal produced no matching candidates")
	}
	return normalized, nil
}

func (runner *Runner) validate(req *proposal.Request) error {
	if runner == nil {
		return fmt.Errorf("replay runner is required")
	}
	if strings.TrimSpace(runner.path) == "" {
		return fmt.Errorf("replay proposal path is required")
	}
	if req == nil {
		return fmt.Errorf("proposal request is required")
	}
	if req.Provider == nil || strings.TrimSpace(req.Provider.ID) == "" {
		return fmt.Errorf("provider is required")
	}
	return nil
}
