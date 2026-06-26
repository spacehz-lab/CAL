package cli

import (
	"context"

	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/observe"
)

// Observer collects CLI provider observations.
type Observer struct{}

// Observe captures CLI help output for a CLI provider.
func (Observer) Observe(ctx context.Context, provider core.Provider) (observe.Result, error) {
	if provider.Kind != core.ProviderKindCLI {
		return observe.Result{ProviderID: provider.ID}, nil
	}
	outputs, err := DocumentationOutputs(ctx, provider.Path)
	if err != nil {
		return observe.Result{ProviderID: provider.ID}, err
	}
	observations := make([]observe.Observation, 0, len(outputs))
	for _, output := range outputs {
		observations = append(observations, observe.Observation{
			Type:   "cli_output",
			Source: output.Source,
			Content: map[string]any{
				"text": output.Text,
			},
		})
	}
	return observe.Result{
		ProviderID:   provider.ID,
		Observations: observations,
	}, nil
}
