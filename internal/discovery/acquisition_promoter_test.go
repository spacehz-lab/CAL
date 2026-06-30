package discovery

import (
	"testing"
	"time"

	"github.com/spacehz-lab/cal/internal/core"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestAcquisitionPromoterRejectsFailedProbe(t *testing.T) {
	promoter := newAcquisitionTestPromoter()
	_, _, err := promoter.promotedCapability(caltrace.Candidate{
		ProviderID:   "provider_cli",
		CapabilityID: "document.convert",
		Description:  "Export a document to PDF.",
		Execution:    core.Execution{Kind: core.ExecutionKindCLI},
	}, caltrace.Probe{Passed: false})
	if err == nil {
		t.Fatal("promotedCapability() error = nil, want failed probe rejection")
	}
}

func TestAcquisitionPromoterRequiresDescription(t *testing.T) {
	promoter := newAcquisitionTestPromoter()
	_, _, err := promoter.promotedCapability(caltrace.Candidate{
		ProviderID:   "provider_cli",
		CapabilityID: "document.convert",
		Execution:    core.Execution{Kind: core.ExecutionKindCLI},
	}, passedFileExistsProbe())
	if err == nil {
		t.Fatal("promotedCapability() error = nil, want description error")
	}
}

func TestAcquisitionPromoterCreatesPromotedBinding(t *testing.T) {
	promoter := newAcquisitionTestPromoter()
	capability, binding, err := promoter.promotedCapability(caltrace.Candidate{
		ProviderID:   "provider_cli",
		CapabilityID: "document.convert",
		Description:  "Export a document to PDF.",
		Execution:    core.Execution{Kind: core.ExecutionKindCLI, Spec: map[string]any{"args": []string{"run"}}},
	}, caltrace.Probe{
		Passed:   true,
		Verify:   fileExistsVerifySpec(),
		Evidence: []core.EvidenceRef{{ID: "evidence_file_exists"}},
	})
	if err != nil {
		t.Fatalf("promotedCapability() error = %v", err)
	}
	if capability.ID != "document.convert" || capability.Description == "" || len(capability.Bindings) != 1 {
		t.Fatalf("capability = %#v, want one promoted binding", capability)
	}
	if binding.State != core.BindingStatePromoted || binding.Verify == nil || len(binding.Evidence) != 1 {
		t.Fatalf("binding = %#v, want promoted binding with verify spec and evidence", binding)
	}
}

func TestAcquisitionPromoterBindingIDUsesExecutionSpec(t *testing.T) {
	promoter := newAcquisitionTestPromoter()
	first, _, err := promoter.promotedCapability(caltrace.Candidate{
		ProviderID:   "provider_cli",
		CapabilityID: "document.convert",
		Description:  "Export a document to PDF.",
		Execution:    core.Execution{Kind: core.ExecutionKindCLI, Spec: map[string]any{"args": []string{"old"}}},
	}, passedFileExistsProbe())
	if err != nil {
		t.Fatalf("promotedCapability() first error = %v", err)
	}
	second, _, err := promoter.promotedCapability(caltrace.Candidate{
		ProviderID:   "provider_cli",
		CapabilityID: "document.convert",
		Description:  "Export a document to PDF.",
		Execution:    core.Execution{Kind: core.ExecutionKindCLI, Spec: map[string]any{"args": []string{"new"}}},
	}, passedFileExistsProbe())
	if err != nil {
		t.Fatalf("promotedCapability() second error = %v", err)
	}
	if first.Bindings[0].ID == second.Bindings[0].ID {
		t.Fatalf("binding id = %q for different executions", first.Bindings[0].ID)
	}
}

func TestAcquisitionPromoterMergeKeepsDifferentBindings(t *testing.T) {
	promoter := newAcquisitionTestPromoter()
	first, _, err := promoter.promotedCapability(caltrace.Candidate{
		ProviderID:   "provider_cli",
		CapabilityID: "document.convert",
		Description:  "Export a document to PDF.",
		Execution:    core.Execution{Kind: core.ExecutionKindCLI, Spec: map[string]any{"args": []string{"old"}}},
	}, passedFileExistsProbe())
	if err != nil {
		t.Fatalf("promotedCapability() first error = %v", err)
	}
	second, _, err := promoter.promotedCapability(caltrace.Candidate{
		ProviderID:   "provider_cli",
		CapabilityID: "document.convert",
		Description:  "Export a document to PDF.",
		Execution:    core.Execution{Kind: core.ExecutionKindCLI, Spec: map[string]any{"args": []string{"new"}}},
	}, passedFileExistsProbe())
	if err != nil {
		t.Fatalf("promotedCapability() second error = %v", err)
	}
	merged := promoter.mergeCapability(first, second)
	if len(merged.Bindings) != 2 {
		t.Fatalf("bindings len = %d, want 2", len(merged.Bindings))
	}
}

func TestAcquisitionPromoterMergeReplacesSameBinding(t *testing.T) {
	promoter := newAcquisitionTestPromoter()
	existing := core.Capability{
		ID:          "document.convert",
		Description: "Export a document to PDF.",
		Bindings: []core.Binding{{
			ID:           "binding_same",
			CapabilityID: "document.convert",
			ProviderID:   "provider_cli",
			Execution:    core.Execution{Kind: core.ExecutionKindCLI, Spec: map[string]any{"args": []string{"old"}}},
			Verify:       fileExistsVerifySpecPtr(),
			Evidence:     []core.EvidenceRef{{ID: "old_evidence"}},
			State:        core.BindingStatePromoted,
		}},
	}
	promoted := core.Capability{
		ID:          "document.convert",
		Description: "Export a document to PDF.",
		Bindings: []core.Binding{{
			ID:           "binding_same",
			CapabilityID: "document.convert",
			ProviderID:   "provider_cli",
			Execution:    core.Execution{Kind: core.ExecutionKindCLI, Spec: map[string]any{"args": []string{"new"}}},
			Verify:       fileExistsVerifySpecPtr(),
			Evidence:     []core.EvidenceRef{{ID: "new_evidence"}},
			State:        core.BindingStatePromoted,
		}},
	}
	merged := promoter.mergeCapability(existing, promoted)
	if len(merged.Bindings) != 1 {
		t.Fatalf("bindings len = %d, want 1", len(merged.Bindings))
	}
	args := merged.Bindings[0].Execution.Spec["args"].([]string)
	if args[0] != "new" {
		t.Fatalf("args = %#v, want replacement binding", args)
	}
}

func newAcquisitionTestPromoter() acquisitionPromoter {
	return newAcquisitionPromoter(nil, core.Provider{ID: "provider_cli"}, time.Unix(0, 0).UTC())
}

func passedFileExistsProbe() caltrace.Probe {
	return caltrace.Probe{
		Passed:   true,
		Verify:   fileExistsVerifySpec(),
		Evidence: []core.EvidenceRef{{ID: "evidence_file_exists"}},
	}
}

func fileExistsVerifySpecPtr() *core.VerifySpec {
	verify := fileExistsVerifySpec()
	return &verify
}

func fileExistsVerifySpec() core.VerifySpec {
	return core.VerifySpec{
		Level:  core.VerifyLevelL2,
		Method: core.VerifyMethodExecute,
		Checks: []core.VerifyCheck{{Subject: core.VerifySubject{Type: core.VerifySubjectFile, Input: "target"}, Predicate: core.VerifyPredicateExists}},
	}
}
