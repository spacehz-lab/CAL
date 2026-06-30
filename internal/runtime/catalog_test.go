package runtime

import (
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

func TestListCapabilitiesSummarizesPromotedBindings(t *testing.T) {
	list := NewCatalog().List([]core.Capability{
		{
			ID:          "document.convert",
			Description: "Export a document to PDF.",
			Bindings: []core.Binding{
				listBinding("binding_b", "provider_b", core.VerifyLevelL3, core.BindingStatePromoted),
				listBinding("binding_a", "provider_a", core.VerifyLevelL2, core.BindingStatePromoted),
				listBinding("binding_draft", "provider_a", core.VerifyLevelL1, ""),
			},
		},
		{
			ID: "document.inspect",
			Bindings: []core.Binding{
				listBinding("binding_unavailable", "provider_a", "ignored", ""),
			},
		},
	}, ListOptions{})

	if list.Count != 1 || len(list.Capabilities) != 1 {
		t.Fatalf("list = %#v, want one available capability", list)
	}

	bindings := list.Capabilities[0].Bindings
	if bindings.Available != 2 {
		t.Fatalf("Available = %d, want 2", bindings.Available)
	}
	if got := bindings.ProviderIDs; len(got) != 2 || got[0] != "provider_a" || got[1] != "provider_b" {
		t.Fatalf("ProviderIDs = %#v, want provider_a then provider_b", got)
	}
	if got := bindings.VerifyLevels; len(got) != 2 || got[0] != "L2" || got[1] != "L3" {
		t.Fatalf("VerifyLevels = %#v, want sorted verify levels", got)
	}
}

func TestListCapabilitiesScopesCapabilityAndProvider(t *testing.T) {
	list := NewCatalog().List([]core.Capability{
		{
			ID: "document.convert",
			Bindings: []core.Binding{
				listBinding("binding_a", "provider_a", core.VerifyLevelL2, core.BindingStatePromoted),
				listBinding("binding_b", "provider_b", core.VerifyLevelL2, core.BindingStatePromoted),
			},
		},
		{
			ID: "document.print",
			Bindings: []core.Binding{
				listBinding("binding_print", "provider_b", core.VerifyLevelL2, core.BindingStatePromoted),
			},
		},
	}, ListOptions{CapabilityID: "document.convert", ProviderID: "provider_b"})
	if list.Count != 1 || len(list.Capabilities) != 1 {
		t.Fatalf("list = %#v, want one selected capability", list)
	}

	bindings := list.Capabilities[0].Bindings
	if list.Capabilities[0].ID != "document.convert" || bindings.Available != 1 || len(bindings.ProviderIDs) != 1 || bindings.ProviderIDs[0] != "provider_b" {
		t.Fatalf("list = %#v, want document.convert provider_b binding only", list)
	}
}

func listBinding(id, providerID string, level core.VerifyLevel, state core.BindingState) core.Binding {
	return core.Binding{
		ID:           id,
		CapabilityID: "document.convert",
		ProviderID:   providerID,
		Execution:    core.Execution{Kind: core.ExecutionKindCLI},
		Verify: &core.VerifySpec{
			Level:  level,
			Checks: []core.VerifyCheck{{Subject: core.VerifySubject{Type: core.VerifySubjectFile, Input: "target"}, Predicate: core.VerifyPredicateExists}},
		},
		State: state,
	}
}
