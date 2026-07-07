package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/spacehz-lab/cal/internal/contract"
	"github.com/spacehz-lab/cal/internal/model"
	runpkg "github.com/spacehz-lab/cal/internal/run"
)

// Run executes one promoted capability binding.
func (app *App) Run(ctx context.Context, req *contract.RunRequest) (*contract.RunResponse, error) {
	if err := app.validate(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, fmt.Errorf("run request is required")
	}
	if err := validateStrategy(req.Strategy); err != nil {
		return nil, err
	}
	started := logOperationStart(ctx, opRun,
		slog.String("provider_id", req.ProviderID),
		slog.String("capability_id", req.CapabilityID),
		slog.String("binding_id", req.BindingID),
	)
	result, err := app.run.Run(ctx, &runpkg.Request{
		CapabilityID:   req.CapabilityID,
		BindingID:      req.BindingID,
		ProviderID:     req.ProviderID,
		Inputs:         req.Inputs,
		Verify:         req.Verify,
		MinVerifyLevel: req.MinVerifyLevel,
	})
	if err != nil {
		logOperationFailure(ctx, opRun, started, err,
			slog.String("provider_id", req.ProviderID),
			slog.String("capability_id", req.CapabilityID),
			slog.String("binding_id", req.BindingID),
		)
		return nil, err
	}
	if result == nil || result.Run == nil {
		err := fmt.Errorf("run result is required")
		logOperationFailure(ctx, opRun, started, err,
			slog.String("provider_id", req.ProviderID),
			slog.String("capability_id", req.CapabilityID),
			slog.String("binding_id", req.BindingID),
		)
		return nil, err
	}
	attrs := []slog.Attr{
		slog.String("run_id", result.Run.ID),
		slog.String("provider_id", result.Run.ProviderID),
		slog.String("capability_id", result.Run.CapabilityID),
		slog.String("binding_id", result.Run.BindingID),
		slog.String("status", string(result.Run.Status)),
	}
	if result.Run.Status == model.RunStatusFailed {
		if result.Run.Error != nil {
			logOperationFailure(ctx, opRun, started, errors.New(result.Run.Error.Message), append(attrs, slog.String("error_code", result.Run.Error.Code))...)
		} else {
			logOperationFailure(ctx, opRun, started, errors.New("run failed"), attrs...)
		}
	} else {
		logOperationSuccess(ctx, opRun, started, attrs...)
	}
	return &contract.RunResponse{Run: result.Run}, nil
}

func validateStrategy(strategy contract.RunStrategy) error {
	switch strategy {
	case "", contract.RunStrategyDefault, contract.RunStrategyFirst, contract.RunStrategyBest:
		return nil
	default:
		return fmt.Errorf("unsupported run strategy %q", strategy)
	}
}
