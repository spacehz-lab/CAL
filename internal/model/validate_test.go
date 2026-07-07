package model

import "testing"

func TestValidateCapabilityAllowsPromotedBindingWithVerifyEvidenceAndExecution(t *testing.T) {
	capability := Capability{
		ID:          "document.convert",
		Description: "Export a document to PDF.",
		Bindings: []Binding{
			{
				ID:           "binding_abc123",
				CapabilityID: "document.convert",
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
		ID:          "document.convert",
		Description: "Export a document to PDF.",
		Bindings: []Binding{
			{
				ID:           "binding_abc123",
				CapabilityID: "document.convert",
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
		ID: "document.convert",
		Bindings: []Binding{
			{
				ID:           "binding_abc123",
				CapabilityID: "document.convert",
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
		ID:          "document.convert",
		Description: "Export a document to PDF.",
		Bindings: []Binding{
			{
				ID:           "binding_abc123",
				CapabilityID: "document.transform",
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
		ID:          "document.convert",
		Description: "Export a document to PDF.",
		Bindings: []Binding{
			{
				ID:           "binding_abc123",
				CapabilityID: "document.convert",
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

func TestValidateProviderRequiresCoreFields(t *testing.T) {
	if err := ValidateProvider(Provider{Kind: ProviderKindCLI, Path: "/tmp/provider"}); err == nil {
		t.Fatal("ValidateProvider() error = nil, want missing id error")
	}
	if err := ValidateProvider(Provider{ID: "provider_abc", Kind: ProviderKindCLI}); err == nil {
		t.Fatal("ValidateProvider() error = nil, want missing path error")
	}
	if err := ValidateProvider(Provider{ID: "provider_abc", Kind: "fake", Path: "/tmp/provider"}); err == nil {
		t.Fatal("ValidateProvider() error = nil, want invalid kind error")
	}
}

func TestValidateRunRequiresCoreFields(t *testing.T) {
	valid := Run{ID: "run_abc", CapabilityID: "document.convert", Status: RunStatusSucceeded}
	if err := ValidateRun(valid); err != nil {
		t.Fatalf("ValidateRun() error = %v", err)
	}
	if err := ValidateRun(Run{CapabilityID: "document.convert", Status: RunStatusSucceeded}); err == nil {
		t.Fatal("ValidateRun() error = nil, want missing id error")
	}
	if err := ValidateRun(Run{ID: "run_abc", Status: RunStatusSucceeded}); err == nil {
		t.Fatal("ValidateRun() error = nil, want missing capability id error")
	}
	if err := ValidateRun(Run{ID: "run_abc", CapabilityID: "document.convert", Status: "done"}); err == nil {
		t.Fatal("ValidateRun() error = nil, want invalid status error")
	}
}

func TestValidateTraceRequiresIDAndKnownStatus(t *testing.T) {
	if err := ValidateTrace(Trace{ID: "trace_abc", Status: TraceStatusCompleted}); err != nil {
		t.Fatalf("ValidateTrace() error = %v", err)
	}
	if err := ValidateTrace(Trace{Status: TraceStatusRunning}); err == nil {
		t.Fatal("ValidateTrace() error = nil, want missing id error")
	}
	if err := ValidateTrace(Trace{ID: "trace_abc", Status: "done"}); err == nil {
		t.Fatal("ValidateTrace() error = nil, want invalid status error")
	}
}

func testVerifySpec() *VerifySpec {
	return &VerifySpec{
		Level:  VerifyLevelL2,
		Method: VerifyMethodExecute,
		Checks: []VerifyCheck{{
			Subject:   VerifySubject{Type: VerifySubjectFile, Input: "target"},
			Predicate: VerifyPredicateExists,
		}},
	}
}
