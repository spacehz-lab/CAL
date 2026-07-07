package eval

import "github.com/spacehz-lab/cal/internal/model"

func aggregateReuse(runs []model.Run) ReuseMetrics {
	metrics := ReuseMetrics{
		Runs:         newCountByStatus(),
		ByProvider:   map[string]RunSlice{},
		ByCapability: map[string]RunSlice{},
	}

	for _, run := range runs {
		addRun(&metrics.Runs, &metrics.Verified, run)
		if run.ProviderID != "" {
			slice := metrics.ByProvider[run.ProviderID]
			addRun(&slice.Runs, &slice.Verified, run)
			metrics.ByProvider[run.ProviderID] = slice
		}
		if run.CapabilityID != "" {
			slice := metrics.ByCapability[run.CapabilityID]
			addRun(&slice.Runs, &slice.Verified, run)
			metrics.ByCapability[run.CapabilityID] = slice
		}
		if run.Error != nil {
			metrics.Errors.add(run.Error.Code)
		}
	}

	return metrics
}

func aggregateCapability(capabilities []model.Capability) CapabilityMetrics {
	var metrics CapabilityMetrics
	for _, capability := range capabilities {
		metrics.Capabilities++
		if len(capability.Bindings) == 0 {
			metrics.CapabilitiesWithoutBindings++
		}
		for _, binding := range capability.Bindings {
			metrics.Bindings++
			if binding.State == model.BindingStatePromoted {
				metrics.PromotedBindings++
			}
			if binding.Verify != nil {
				metrics.BindingsWithVerify++
			}
		}
	}
	return metrics
}

func addRun(counts *CountByStatus, verified *int, run model.Run) {
	if counts.ByName == nil {
		*counts = newCountByStatus()
	}
	counts.add(string(run.Status))
	if run.Verified {
		(*verified)++
	}
}
