package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/spacehz-lab/cal/internal/acquisition"
	"github.com/spacehz-lab/cal/internal/contract"
	"github.com/spacehz-lab/cal/internal/model"
)

// Acquire runs one acquisition flow.
func (app *App) Acquire(ctx context.Context, req *contract.AcquisitionRequest) (*contract.AcquisitionResponse, error) {
	if err := app.validate(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, fmt.Errorf("acquisition request is required")
	}
	started := logOperationStart(ctx, opAcquire,
		slog.String("provider_id", req.ProviderID),
		slog.String("hint", req.Hint),
		slog.String("mode", string(acquisitionMode(req.Mode))),
	)
	proposer, err := app.proposerFor(req)
	if err != nil {
		logOperationFailure(ctx, opAcquire, started, err,
			slog.String("provider_id", req.ProviderID),
			slog.String("hint", req.Hint),
		)
		return nil, err
	}
	runner := app.acquire.runner(proposer)
	if runner == nil {
		err := fmt.Errorf("acquisition runner is not configured")
		logOperationFailure(ctx, opAcquire, started, err,
			slog.String("provider_id", req.ProviderID),
			slog.String("hint", req.Hint),
		)
		return nil, err
	}

	result, err := runner.Run(ctx, &acquisition.Request{
		ProviderID: req.ProviderID,
		Hint:       req.Hint,
		WorkRoot:   app.workRoot,
	})
	if result != nil {
		response := acquisitionResponse(result)
		if response != nil {
			attrs := []slog.Attr{
				slog.String("trace_id", response.TraceID),
				slog.String("provider_id", req.ProviderID),
				slog.String("hint", req.Hint),
			}
			if response.Error != nil {
				logOperationFailure(ctx, opAcquire, started, errors.New(response.Error.Message), append(attrs, slog.String("error_code", response.Error.Code))...)
			} else if response.Trace != nil && response.Trace.Status != model.TraceStatusCompleted {
				logOperationFailure(ctx, opAcquire, started, fmt.Errorf("acquisition status %s", response.Trace.Status), append(attrs, slog.String("status", string(response.Trace.Status)))...)
			} else {
				logOperationSuccess(ctx, opAcquire, started, append(attrs,
					slog.Int("capabilities_promoted", response.CapabilitiesPromoted),
					slog.Int("bindings_promoted", response.BindingsPromoted),
				)...)
			}
			return response, nil
		}
	}
	if err != nil {
		logOperationFailure(ctx, opAcquire, started, err,
			slog.String("provider_id", req.ProviderID),
			slog.String("hint", req.Hint),
		)
	}
	return nil, err
}

func acquisitionMode(mode contract.AcquisitionMode) contract.AcquisitionMode {
	if mode == "" {
		return contract.AcquisitionModeLive
	}
	return mode
}
