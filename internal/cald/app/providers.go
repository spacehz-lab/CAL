package app

import (
	"context"

	"github.com/spacehz-lab/cal/internal/contract"
	"github.com/spacehz-lab/cal/internal/entry"
)

// AddProvider registers one provider path and returns the current provider list.
func (app *App) AddProvider(ctx context.Context, req *contract.AddProviderRequest) (*contract.ProviderListResponse, error) {
	if err := app.validate(); err != nil {
		return nil, err
	}
	if req == nil {
		req = &contract.AddProviderRequest{}
	}
	if _, err := app.registry.Register(ctx, &entry.RegisterRequest{ProviderPath: req.ProviderPath}); err != nil {
		return nil, err
	}
	return app.ListProviders(ctx)
}

// ListProviders returns registered providers.
func (app *App) ListProviders(ctx context.Context) (*contract.ProviderListResponse, error) {
	if err := app.validate(); err != nil {
		return nil, err
	}
	providers, err := app.registry.List(ctx)
	if err != nil {
		return nil, err
	}
	return &contract.ProviderListResponse{Providers: providers}, nil
}
