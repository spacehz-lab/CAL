package runtime

import (
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

func TestListCapabilitiesSummarizesPromotedBindings(t *testing.T) {
	list := NewCatalog().List([]core.Capability{
		{
			ID:          "document.export_pdf",
			Description: "Export a document to PDF.",
			Bindings: []core.Binding{
				listBinding("binding_b", "provider_b", "pdf_exists", core.BindingStatePromoted),
				listBinding("binding_a", "provider_a", "file_exists", core.BindingStatePromoted),
				listBinding("binding_draft", "provider_a", "ignored", ""),
			},
		},
		{
			ID: "document.no_promoted_binding",
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
	if got := bindings.Verifiers; len(got) != 2 || got[0] != "file_exists" || got[1] != "pdf_exists" {
		t.Fatalf("Verifiers = %#v, want sorted verifier ids", got)
	}
}

func TestListCapabilitiesScopesCapabilityAndProvider(t *testing.T) {
	list := NewCatalog().List([]core.Capability{
		{
			ID: "document.export_pdf",
			Bindings: []core.Binding{
				listBinding("binding_a", "provider_a", "file_exists", core.BindingStatePromoted),
				listBinding("binding_b", "provider_b", "file_exists", core.BindingStatePromoted),
			},
		},
		{
			ID: "document.print",
			Bindings: []core.Binding{
				listBinding("binding_print", "provider_b", "printed", core.BindingStatePromoted),
			},
		},
	}, ListOptions{CapabilityID: "document.export_pdf", ProviderID: "provider_b"})
	if list.Count != 1 || len(list.Capabilities) != 1 {
		t.Fatalf("list = %#v, want one selected capability", list)
	}

	bindings := list.Capabilities[0].Bindings
	if list.Capabilities[0].ID != "document.export_pdf" || bindings.Available != 1 || len(bindings.ProviderIDs) != 1 || bindings.ProviderIDs[0] != "provider_b" {
		t.Fatalf("list = %#v, want document.export_pdf provider_b binding only", list)
	}
}

func listBinding(id, providerID, verifierType string, state core.BindingState) core.Binding {
	return core.Binding{
		ID:           id,
		CapabilityID: "document.export_pdf",
		ProviderID:   providerID,
		Execution:    core.Execution{Kind: core.ExecutionKindCLI},
		Verifier:     &core.Verifier{ID: verifierType},
		State:        state,
	}
}
