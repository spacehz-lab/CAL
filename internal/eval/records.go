package eval

import (
	"context"
	"fmt"

	"github.com/spacehz-lab/cal/internal/model"
)

type records struct {
	Traces       []model.Trace
	Runs         []model.Run
	Capabilities []model.Capability
}

type recordFilter struct {
	ProviderID   string
	CapabilityID string
}

func (runner *Runner) loadRecords(ctx context.Context, filter recordFilter) (*records, error) {
	traces, err := runner.store.ListTraces()
	if err != nil {
		return nil, fmt.Errorf("list traces: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	runs, err := runner.store.ListRuns()
	if err != nil {
		return nil, fmt.Errorf("list runs: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	capabilities, err := runner.store.ListCapabilities()
	if err != nil {
		return nil, fmt.Errorf("list capabilities: %w", err)
	}

	return &records{
		Traces:       filterTraces(traces, filter),
		Runs:         filterRuns(runs, filter),
		Capabilities: filterCapabilities(capabilities, filter),
	}, nil
}

func filterTraces(traces []model.Trace, filter recordFilter) []model.Trace {
	if filter.ProviderID == "" && filter.CapabilityID == "" {
		return traces
	}
	filtered := make([]model.Trace, 0, len(traces))
	for _, trace := range traces {
		if traceMatches(trace, filter) {
			filtered = append(filtered, trace)
		}
	}
	return filtered
}

func filterRuns(runs []model.Run, filter recordFilter) []model.Run {
	if filter.ProviderID == "" && filter.CapabilityID == "" {
		return runs
	}
	filtered := make([]model.Run, 0, len(runs))
	for _, run := range runs {
		if filter.ProviderID != "" && run.ProviderID != filter.ProviderID {
			continue
		}
		if filter.CapabilityID != "" && run.CapabilityID != filter.CapabilityID {
			continue
		}
		filtered = append(filtered, run)
	}
	return filtered
}

func filterCapabilities(capabilities []model.Capability, filter recordFilter) []model.Capability {
	if filter.ProviderID == "" && filter.CapabilityID == "" {
		return capabilities
	}
	filtered := make([]model.Capability, 0, len(capabilities))
	for _, capability := range capabilities {
		if filter.CapabilityID != "" && capability.ID != filter.CapabilityID {
			continue
		}
		if filter.ProviderID != "" && !capabilityHasProvider(capability, filter.ProviderID) {
			continue
		}
		filtered = append(filtered, capability)
	}
	return filtered
}

func traceMatches(trace model.Trace, filter recordFilter) bool {
	if filter.ProviderID != "" && !traceHasProvider(trace, filter.ProviderID) {
		return false
	}
	if filter.CapabilityID != "" && !traceHasCapability(trace, filter.CapabilityID) {
		return false
	}
	return true
}

func traceHasProvider(trace model.Trace, providerID string) bool {
	for _, id := range trace.ProviderIDs {
		if id == providerID {
			return true
		}
	}
	for _, observation := range trace.Observations {
		if observation.ProviderID == providerID {
			return true
		}
	}
	for _, candidate := range trace.Candidates {
		if candidate.ProviderID == providerID {
			return true
		}
	}
	for _, promotion := range trace.Promotions {
		if promotion.ProviderID == providerID {
			return true
		}
	}
	return false
}

func traceHasCapability(trace model.Trace, capabilityID string) bool {
	for _, candidate := range trace.Candidates {
		if candidate.CapabilityID == capabilityID {
			return true
		}
	}
	for _, promotion := range trace.Promotions {
		if promotion.CapabilityID == capabilityID {
			return true
		}
	}
	return false
}

func capabilityHasProvider(capability model.Capability, providerID string) bool {
	for _, binding := range capability.Bindings {
		if binding.ProviderID == providerID {
			return true
		}
	}
	return false
}
