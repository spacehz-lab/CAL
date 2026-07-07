package cli

import (
	"context"

	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/observe"
)

// Observer collects CLI usage observations.
type Observer struct{}

// NewObserver creates a CLI usage observer.
func NewObserver() *Observer {
	return &Observer{}
}

// Observe captures CLI usage output for one CLI provider.
func (observer *Observer) Observe(ctx context.Context, req *observe.Request) (*observe.Result, error) {
	if req == nil || req.Provider == nil {
		return &observe.Result{}, nil
	}
	provider := req.Provider
	if provider.Kind != model.ProviderKindCLI {
		return &observe.Result{ProviderID: provider.ID}, nil
	}

	outputs, err := UsageOutputs(ctx, provider.Path)
	if err != nil {
		return &observe.Result{ProviderID: provider.ID}, err
	}
	observations := make([]model.Observation, 0, len(outputs))
	for _, output := range outputs {
		observations = append(observations, model.Observation{
			ProviderID: provider.ID,
			Type:       observe.ObservationTypeCLIOutput,
			Source:     output.Source,
			Content: map[string]any{
				observe.ObservationContentText: output.Text,
			},
		})
	}
	return &observe.Result{
		ProviderID:   provider.ID,
		Observations: observations,
	}, nil
}
