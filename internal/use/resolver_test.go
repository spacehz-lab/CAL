package use

import (
	"context"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

func TestRequestValidate(t *testing.T) {
	if err := (Request{Inputs: map[string]any{}}).Validate(); err == nil || err.Code != CodeInvalidInput {
		t.Fatalf("Validate() = %#v, want invalid input", err)
	}
	if err := (Request{Intent: "export pdf"}).Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestResolverSelectsPromotedBinding(t *testing.T) {
	capabilities := []core.Capability{
		testCapability("document.export_pdf", "Export a document to PDF.", "binding_pdf", "provider_a", []string{"export-pdf"}),
		testCapability("image.resize", "Resize an image.", "binding_resize", "provider_a", []string{"resize"}),
	}

	selection, err := NewResolver(Request{
		Intent: "export this document as pdf",
		Inputs: map[string]any{},
	}).Select(context.Background(), capabilities)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if selection.CapabilityID != "document.export_pdf" || selection.BindingID != "binding_pdf" || selection.ProviderID != "provider_a" {
		t.Fatalf("Select() = %#v, want PDF binding", selection)
	}
}

func TestResolverSelectsBindingWithMissingInputs(t *testing.T) {
	capabilities := []core.Capability{
		testCapability("document.export_pdf", "Export a document to PDF.", "binding_pdf", "provider_a", []string{"export-pdf", "--source", "{{source}}", "--target", "{{target}}"}),
	}

	selection, err := NewResolver(Request{
		Intent: "export this document as pdf",
		Inputs: map[string]any{"source": "input.md"},
	}).Select(context.Background(), capabilities)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if selection.BindingID != "binding_pdf" {
		t.Fatalf("Select() = %#v, want binding_pdf", selection)
	}
}

func TestResolverReturnsAmbiguousMatch(t *testing.T) {
	capabilities := []core.Capability{
		testCapability("document.export_pdf", "", "binding_a", "provider_a", []string{"export-pdf"}),
		testCapability("pdf.export_document", "", "binding_b", "provider_b", []string{"make-pdf"}),
	}

	_, err := NewResolver(Request{
		Intent: "export pdf",
		Inputs: map[string]any{},
	}).Select(context.Background(), capabilities)
	if err == nil || err.Code != CodeAmbiguous {
		t.Fatalf("Select() error = %#v, want ambiguous", err)
	}
}

func testCapability(id, description, bindingID, providerID string, args []string) core.Capability {
	return core.Capability{
		ID:          id,
		Description: description,
		Bindings: []core.Binding{{
			ID:           bindingID,
			CapabilityID: id,
			ProviderID:   providerID,
			Execution: core.Execution{
				Kind: core.ExecutionKindCLI,
				Spec: map[string]any{core.ExecutionSpecArgs: args},
			},
			Verify: &core.VerifySpec{
				Level:  core.VerifyLevelL2,
				Method: core.VerifyMethodExecute,
				Checks: []core.VerifyCheck{{Subject: "target", Predicate: core.VerifyPredicateExists}},
			},
			State: core.BindingStatePromoted,
		}},
	}
}
