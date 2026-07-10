package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/spacehz-lab/cal/internal/contract"
	"github.com/spacehz-lab/cal/internal/model"
	usepkg "github.com/spacehz-lab/cal/internal/use"
)

// Use selects a promoted binding from intent and delegates to run.
func (app *App) Use(ctx context.Context, req *contract.UseRequest) (*contract.UseResponse, error) {
	if err := app.validate(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, fmt.Errorf("use request is required")
	}
	if err := validateStrategy(req.Strategy); err != nil {
		return nil, err
	}
	started := logOperationStart(ctx, opUse,
		slog.String("provider_id", req.ProviderID),
	)
	result, err := app.use.Run(ctx, &usepkg.Request{
		Intent:            req.Intent,
		Inputs:            req.Inputs,
		ProviderID:        req.ProviderID,
		ForceLLMSelection: req.Strategy == contract.RunStrategyBest,
		Verify:            req.Verify,
		MinVerifyLevel:    req.MinVerifyLevel,
	})
	if err != nil {
		logOperationFailure(ctx, opUse, started, err, slog.String("provider_id", req.ProviderID))
		return nil, err
	}
	if result == nil {
		err := fmt.Errorf("use result is required")
		logOperationFailure(ctx, opUse, started, err, slog.String("provider_id", req.ProviderID))
		return nil, err
	}
	attrs := []slog.Attr{
		slog.String("use_id", result.ID),
		slog.String("status", string(result.Status)),
	}
	if result.Selection != nil {
		attrs = append(attrs,
			slog.String("provider_id", result.Selection.ProviderID),
			slog.String("capability_id", result.Selection.CapabilityID),
			slog.String("binding_id", result.Selection.BindingID),
		)
	}
	if result.Run != nil {
		attrs = append(attrs, slog.String("run_id", result.Run.ID))
	}
	if result.Status == model.RunStatusFailed {
		if result.Error != nil {
			logOperationFailure(ctx, opUse, started, errors.New(result.Error.Message), append(attrs, slog.String("error_code", result.Error.Code))...)
		} else {
			logOperationFailure(ctx, opUse, started, errors.New("use failed"), attrs...)
		}
	} else {
		logOperationSuccess(ctx, opUse, started, attrs...)
	}
	return useResponse(result), nil
}
