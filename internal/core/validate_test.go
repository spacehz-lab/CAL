package core

import "testing"

func TestValidateCapabilityAllowsPromotedBindingWithVerifierEvidenceAndExecution(t *testing.T) {
	capability := Capability{
		ID:          "document.export_pdf",
		Description: "Export a document to PDF.",
		Bindings: []Binding{
			{
				ID:           "binding_abc123",
				CapabilityID: "document.export_pdf",
				ProviderID:   "provider_abc123",
				Execution:    Execution{Kind: ExecutionKindCLI},
				Verifier:     &Verifier{ID: "file_exists"},
				Evidence:     []EvidenceRef{{ID: "evidence_abc123"}},
				State:        BindingStatePromoted,
			},
		},
	}

	if err := ValidateCapability(capability); err != nil {
		t.Fatalf("ValidateCapability() error = %v", err)
	}
}

func TestValidateCapabilityRequiresPromotedVerifierAndEvidence(t *testing.T) {
	capability := Capability{
		ID:          "document.export_pdf",
		Description: "Export a document to PDF.",
		Bindings: []Binding{
			{
				ID:           "binding_abc123",
				CapabilityID: "document.export_pdf",
				ProviderID:   "provider_abc123",
				Execution:    Execution{Kind: ExecutionKindCLI},
				State:        BindingStatePromoted,
			},
		},
	}

	if err := ValidateCapability(capability); err == nil {
		t.Fatal("ValidateCapability() error = nil, want promoted binding verifier error")
	}
}

func TestValidateCapabilityRequiresDescriptionForPromotedBinding(t *testing.T) {
	capability := Capability{
		ID: "document.export_pdf",
		Bindings: []Binding{
			{
				ID:           "binding_abc123",
				CapabilityID: "document.export_pdf",
				ProviderID:   "provider_abc123",
				Execution:    Execution{Kind: ExecutionKindCLI},
				Verifier:     &Verifier{ID: "file_exists"},
				Evidence:     []EvidenceRef{{ID: "evidence_abc123"}},
				State:        BindingStatePromoted,
			},
		},
	}

	if err := ValidateCapability(capability); err == nil {
		t.Fatal("ValidateCapability() error = nil, want promoted capability description error")
	}
}

func TestValidateCapabilityRejectsMismatchedBindingCapability(t *testing.T) {
	capability := Capability{
		ID:          "document.export_pdf",
		Description: "Export a document to PDF.",
		Bindings: []Binding{
			{
				ID:           "binding_abc123",
				CapabilityID: "document.replace_text",
				ProviderID:   "provider_abc123",
				Execution:    Execution{Kind: ExecutionKindCLI},
				Verifier:     &Verifier{ID: "file_exists"},
				Evidence:     []EvidenceRef{{ID: "evidence_abc123"}},
				State:        BindingStatePromoted,
			},
		},
	}

	if err := ValidateCapability(capability); err == nil {
		t.Fatal("ValidateCapability() error = nil, want mismatch error")
	}
}

func TestValidateCapabilityRejectsInvalidExecutionKind(t *testing.T) {
	capability := Capability{
		ID:          "document.export_pdf",
		Description: "Export a document to PDF.",
		Bindings: []Binding{
			{
				ID:           "binding_abc123",
				CapabilityID: "document.export_pdf",
				ProviderID:   "provider_abc123",
				Execution:    Execution{Kind: "direct_file"},
				Verifier:     &Verifier{ID: "file_exists"},
				Evidence:     []EvidenceRef{{ID: "evidence_abc123"}},
				State:        BindingStatePromoted,
			},
		},
	}

	if err := ValidateCapability(capability); err == nil {
		t.Fatal("ValidateCapability() error = nil, want invalid execution kind error")
	}
}

func TestValidateCapabilityRejectsInvalidVerifierID(t *testing.T) {
	capability := Capability{
		ID:          "document.export_pdf",
		Description: "Export a document to PDF.",
		Bindings: []Binding{
			{
				ID:           "binding_abc123",
				CapabilityID: "document.export_pdf",
				ProviderID:   "provider_abc123",
				Execution:    Execution{Kind: ExecutionKindCLI},
				Verifier:     &Verifier{ID: "provider.fake"},
				Evidence:     []EvidenceRef{{ID: "evidence_abc123"}},
				State:        BindingStatePromoted,
			},
		},
	}

	if err := ValidateCapability(capability); err == nil {
		t.Fatal("ValidateCapability() error = nil, want invalid verifier id error")
	}
}

func TestValidateProviderRequiresID(t *testing.T) {
	if err := ValidateProvider(Provider{Kind: ProviderKindCLI, Path: "/tmp/provider"}); err == nil {
		t.Fatal("ValidateProvider() error = nil, want missing id error")
	}
}

func TestValidateProviderRequiresPath(t *testing.T) {
	if err := ValidateProvider(Provider{ID: "provider_abc", Kind: ProviderKindCLI}); err == nil {
		t.Fatal("ValidateProvider() error = nil, want missing path error")
	}
}

func TestValidateProviderRejectsInvalidKind(t *testing.T) {
	if err := ValidateProvider(Provider{ID: "provider_abc", Kind: "fake", Path: "/tmp/provider"}); err == nil {
		t.Fatal("ValidateProvider() error = nil, want invalid kind error")
	}
}
