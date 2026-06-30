package runtime

import (
	"context"

	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/verify"
)

func evaluateVerifySpec(ctx context.Context, spec core.VerifySpec, inputs map[string]any, result ExecutionResult) ([]core.EvidenceRef, map[string]any, error) {
	return verify.Evaluate(ctx, spec, verify.Context{
		Inputs:   inputs,
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		ExitCode: result.ExitCode,
	})
}
