package eval

import "github.com/spacehz-lab/cal/internal/model"

func aggregateAcquisition(traces []model.Trace) AcquisitionMetrics {
	metrics := AcquisitionMetrics{Traces: newCountByStatus()}
	promotedCapabilities := map[string]struct{}{}
	promotedBindings := map[string]struct{}{}

	for _, trace := range traces {
		metrics.Traces.add(string(trace.Status))
		metrics.Candidates += len(trace.Candidates)
		metrics.Probes.Total += len(trace.Probes)
		for _, probe := range trace.Probes {
			if probe.Passed {
				metrics.Probes.Passed++
			} else {
				metrics.Probes.Failed++
			}
			if probe.Error != nil {
				metrics.Errors.add(probe.Error.Code)
			}
		}
		for _, promotion := range trace.Promotions {
			metrics.Promotions.Total++
			if promotion.CapabilityID != "" {
				promotedCapabilities[promotion.CapabilityID] = struct{}{}
			}
			if promotion.BindingID != "" {
				promotedBindings[promotion.BindingID] = struct{}{}
			}
		}
		if trace.Error != nil {
			metrics.Errors.add(trace.Error.Code)
		}
	}

	metrics.Promotions.Capabilities = len(promotedCapabilities)
	metrics.Promotions.Bindings = len(promotedBindings)
	return metrics
}
