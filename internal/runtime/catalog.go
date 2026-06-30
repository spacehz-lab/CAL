package runtime

import (
	"sort"

	"github.com/spacehz-lab/cal/internal/core"
)

// CapabilityList is the JSON-safe read model returned by capability list.
type CapabilityList struct {
	Count        int                 `json:"count"`
	Capabilities []CapabilitySummary `json:"capabilities"`
}

// CapabilitySummary describes reusable capability metadata without execution specs.
type CapabilitySummary struct {
	ID          string         `json:"id"`
	Description string         `json:"description,omitempty"`
	Bindings    BindingSummary `json:"bindings"`
}

// BindingSummary describes promoted binding availability for one capability.
type BindingSummary struct {
	Available    int      `json:"available"`
	ProviderIDs  []string `json:"provider_ids"`
	VerifyLevels []string `json:"verify_levels"`
}

// Catalog builds read models for reusable capabilities.
type Catalog struct{}

// ListOptions filters the capability catalog view.
type ListOptions struct {
	CapabilityID string
	ProviderID   string
}

// NewCatalog builds a capability catalog reader.
func NewCatalog() Catalog {
	return Catalog{}
}

// List builds the capability list view from stored capabilities.
func (catalog Catalog) List(capabilities []core.Capability, opts ListOptions) CapabilityList {
	summaries := make([]CapabilitySummary, 0, len(capabilities))
	for _, capability := range capabilities {
		if opts.CapabilityID != "" && capability.ID != opts.CapabilityID {
			continue
		}
		summary, ok := summarizeCapability(capability, opts.ProviderID)
		if ok {
			summaries = append(summaries, summary)
		}
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].ID < summaries[j].ID
	})
	return CapabilityList{Count: len(summaries), Capabilities: summaries}
}

func summarizeCapability(capability core.Capability, providerID string) (CapabilitySummary, bool) {
	providerIDs := map[string]struct{}{}
	verifyLevels := map[string]struct{}{}
	available := 0

	for _, binding := range capability.Bindings {
		if binding.State != core.BindingStatePromoted {
			continue
		}
		if providerID != "" && binding.ProviderID != providerID {
			continue
		}
		available++
		providerIDs[binding.ProviderID] = struct{}{}
		if binding.Verify != nil {
			verifyLevels[string(binding.Verify.Level)] = struct{}{}
		}
	}
	if available == 0 {
		return CapabilitySummary{}, false
	}

	return CapabilitySummary{
		ID:          capability.ID,
		Description: capability.Description,
		Bindings: BindingSummary{
			Available:    available,
			ProviderIDs:  sortedStrings(providerIDs),
			VerifyLevels: sortedStrings(verifyLevels),
		},
	}, true
}

func sortedStrings(values map[string]struct{}) []string {
	sorted := make([]string, 0, len(values))
	for value := range values {
		sorted = append(sorted, value)
	}
	sort.Strings(sorted)
	return sorted
}
