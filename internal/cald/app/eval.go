package app

import (
	"context"

	"github.com/spacehz-lab/cal/internal/contract"
	evalpkg "github.com/spacehz-lab/cal/internal/eval"
)

// Eval returns read-only acquisition and reuse metrics.
func (app *App) Eval(ctx context.Context, req *contract.EvalRequest) (*contract.EvalResponse, error) {
	if err := app.validate(); err != nil {
		return nil, err
	}
	if req == nil {
		req = &contract.EvalRequest{}
	}
	result, err := app.eval.Run(ctx, &evalpkg.Request{
		ProviderID:   req.ProviderID,
		CapabilityID: req.CapabilityID,
	})
	if err != nil {
		return nil, err
	}
	return evalResponse(result), nil
}
