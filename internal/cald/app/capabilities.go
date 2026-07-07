package app

import (
	"context"
	"sort"
	"strings"

	"github.com/spacehz-lab/cal/internal/contract"
	"github.com/spacehz-lab/cal/internal/model"
)

// ListCapabilities returns compact promoted capability summaries.
func (app *App) ListCapabilities(ctx context.Context, req *contract.CapabilityListRequest) (*contract.CapabilityListResponse, error) {
	if err := app.validate(); err != nil {
		return nil, err
	}
	if err := ctxErr(ctx); err != nil {
		return nil, err
	}
	capabilities, err := app.store.ListCapabilities()
	if err != nil {
		return nil, err
	}
	filter := capabilityFilter(req)
	summaries := make([]contract.CapabilitySummary, 0, len(capabilities))
	for _, capability := range capabilities {
		summary, ok := capabilitySummary(capability, filter)
		if ok {
			summaries = append(summaries, summary)
		}
	}
	return &contract.CapabilityListResponse{Count: len(summaries), Capabilities: summaries}, nil
}

type capabilityListFilter struct {
	capabilityID string
	providerID   string
}

func capabilityFilter(req *contract.CapabilityListRequest) capabilityListFilter {
	if req == nil {
		return capabilityListFilter{}
	}
	return capabilityListFilter{
		capabilityID: strings.TrimSpace(req.CapabilityID),
		providerID:   strings.TrimSpace(req.ProviderID),
	}
}

func capabilitySummary(capability model.Capability, filter capabilityListFilter) (contract.CapabilitySummary, bool) {
	if filter.capabilityID != "" && capability.ID != filter.capabilityID {
		return contract.CapabilitySummary{}, false
	}
	bindings := promotedBindings(capability.Bindings, filter.providerID)
	if filter.providerID != "" && len(bindings) == 0 {
		return contract.CapabilitySummary{}, false
	}
	return contract.CapabilitySummary{
		ID:          capability.ID,
		Description: capability.Description,
		Bindings:    bindingSummary(bindings),
	}, true
}

func promotedBindings(bindings []model.Binding, providerID string) []model.Binding {
	result := make([]model.Binding, 0, len(bindings))
	for _, binding := range bindings {
		if binding.State != model.BindingStatePromoted {
			continue
		}
		if providerID != "" && binding.ProviderID != providerID {
			continue
		}
		result = append(result, binding)
	}
	return result
}

func bindingSummary(bindings []model.Binding) contract.BindingSummary {
	providers := map[string]struct{}{}
	levels := map[string]struct{}{}
	for _, binding := range bindings {
		if binding.ProviderID != "" {
			providers[binding.ProviderID] = struct{}{}
		}
		if binding.Verify != nil && binding.Verify.Level != "" {
			levels[string(binding.Verify.Level)] = struct{}{}
		}
	}
	return contract.BindingSummary{
		Available:    len(bindings),
		ProviderIDs:  sortedKeys(providers),
		VerifyLevels: sortedKeys(levels),
	}
}

func sortedKeys(values map[string]struct{}) []string {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
