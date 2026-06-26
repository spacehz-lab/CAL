package eval

import "github.com/spacehz-lab/cal/internal/core"

func (records records) summary() SummaryMetrics {
	metrics := SummaryMetrics{
		Providers:    len(records.providers),
		Capabilities: len(records.capabilities),
		Traces:       len(records.traces),
		Runs:         len(records.runs),
	}
	for _, capability := range records.capabilities {
		for _, binding := range capability.Bindings {
			metrics.Bindings++
			if binding.State == core.BindingStatePromoted {
				metrics.PromotedBindings++
			}
		}
	}
	return metrics
}
