package core

import "testing"

func TestValidateCapabilityAllowsPromotedBindingWithVerifyEvidenceAndExecution(t *testing.T) {
	capability := Capability{
		ID:          "document.export_pdf",
		Description: "Export a document to PDF.",
		Bindings: []Binding{
			{
				ID:           "binding_abc123",
				CapabilityID: "document.export_pdf",
				ProviderID:   "provider_abc123",
				Execution:    Execution{Kind: ExecutionKindCLI},
				Verify:       testVerifySpec(),
				Evidence:     []EvidenceRef{{ID: "evidence_abc123"}},
				State:        BindingStatePromoted,
			},
		},
	}

	if err := ValidateCapability(capability); err != nil {
		t.Fatalf("ValidateCapability() error = %v", err)
	}
}

func TestValidateCapabilityRequiresPromotedVerifyAndEvidence(t *testing.T) {
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
		t.Fatal("ValidateCapability() error = nil, want promoted binding verify error")
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
				Verify:       testVerifySpec(),
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
				Verify:       testVerifySpec(),
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
				Verify:       testVerifySpec(),
				Evidence:     []EvidenceRef{{ID: "evidence_abc123"}},
				State:        BindingStatePromoted,
			},
		},
	}

	if err := ValidateCapability(capability); err == nil {
		t.Fatal("ValidateCapability() error = nil, want invalid execution kind error")
	}
}

func TestValidateCapabilityRejectsInvalidVerifySpec(t *testing.T) {
	capability := Capability{
		ID:          "document.export_pdf",
		Description: "Export a document to PDF.",
		Bindings: []Binding{
			{
				ID:           "binding_abc123",
				CapabilityID: "document.export_pdf",
				ProviderID:   "provider_abc123",
				Execution:    Execution{Kind: ExecutionKindCLI},
				Verify:       &VerifySpec{Level: VerifyLevelL2},
				Evidence:     []EvidenceRef{{ID: "evidence_abc123"}},
				State:        BindingStatePromoted,
			},
		},
	}

	if err := ValidateCapability(capability); err == nil {
		t.Fatal("ValidateCapability() error = nil, want invalid verify spec error")
	}
}

func TestValidateVerifySpecRejectsContractAboveL1(t *testing.T) {
	verify := VerifySpec{Level: VerifyLevelL2, Method: VerifyMethodContract}
	if err := ValidateVerifySpec(verify); err == nil {
		t.Fatal("ValidateVerifySpec() error = nil, want contract level error")
	}
}

func TestValidateVerifySpecRejectsContractChecks(t *testing.T) {
	verify := VerifySpec{
		Level:  VerifyLevelL1,
		Method: VerifyMethodContract,
		Checks: []VerifyCheck{{Subject: "stdout", Predicate: VerifyPredicateNonEmpty}},
	}
	if err := ValidateVerifySpec(verify); err == nil {
		t.Fatal("ValidateVerifySpec() error = nil, want contract checks error")
	}
}

func TestValidateVerifySpecRequiresExecuteChecksExceptL0(t *testing.T) {
	if err := ValidateVerifySpec(VerifySpec{Level: VerifyLevelL1, Method: VerifyMethodExecute}); err == nil {
		t.Fatal("ValidateVerifySpec() error = nil, want execute checks error")
	}
	if err := ValidateVerifySpec(VerifySpec{Level: VerifyLevelL0, Method: VerifyMethodExecute}); err != nil {
		t.Fatalf("ValidateVerifySpec(L0 execute) error = %v", err)
	}
}

func TestValidateVerifySpecRejectsMissingPredicateParams(t *testing.T) {
	verify := VerifySpec{
		Level:  VerifyLevelL2,
		Method: VerifyMethodExecute,
		Checks: []VerifyCheck{{Subject: "exit_code", Predicate: VerifyPredicateEquals, Params: map[string]any{"expected": 0}}},
	}
	if err := ValidateVerifySpec(verify); err == nil {
		t.Fatal("ValidateVerifySpec() error = nil, want missing params.value error")
	}
}

func TestValidateVerifySpecRejectsWrongPredicateParamKey(t *testing.T) {
	verify := VerifySpec{
		Level:  VerifyLevelL2,
		Method: VerifyMethodExecute,
		Checks: []VerifyCheck{{Subject: "stdout", Predicate: VerifyPredicateContains, Params: map[string]any{"expected": "ok"}}},
	}
	if err := ValidateVerifySpec(verify); err == nil {
		t.Fatal("ValidateVerifySpec() error = nil, want wrong params key error")
	}
}

func TestValidateVerifySpecRejectsInvalidRegexPattern(t *testing.T) {
	verify := VerifySpec{
		Level:  VerifyLevelL2,
		Method: VerifyMethodExecute,
		Checks: []VerifyCheck{{Subject: "stdout", Predicate: VerifyPredicateRegex, Params: map[string]any{"pattern": "["}}},
	}
	if err := ValidateVerifySpec(verify); err == nil {
		t.Fatal("ValidateVerifySpec() error = nil, want invalid regex error")
	}
}

func testVerifySpec() *VerifySpec {
	return &VerifySpec{
		Level:  VerifyLevelL2,
		Method: VerifyMethodExecute,
		Checks: []VerifyCheck{{
			Subject:   "target",
			Predicate: VerifyPredicateExists,
		}},
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
