package proposalflow

import (
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestNormalizeCapabilityStageFiltersAndMerges(t *testing.T) {
	policy := CapabilityPolicy{
		PreferredSubjects:   []string{"document", "text"},
		PreferredOperations: []string{"convert", "encode"},
	}
	surfaces := []surface{
		{ID: "s1", Name: "convert"},
		{ID: "s2", Name: "export"},
		{ID: "s3", Name: "encode"},
	}
	input := []capabilityPlan{
		{CapabilityID: "document.convert", SourceSurfaceIDs: []string{"s1"}},
		{CapabilityID: "document.convert", SourceSurfaceIDs: []string{"s2"}},
		{CapabilityID: "document.export_pdf", SourceSurfaceIDs: []string{"s2"}},
		{CapabilityID: "text.encode", SourceSurfaceIDs: []string{"missing"}},
	}

	items, stage := normalizeCapabilities(input, policy, surfaces, Request{}, profile{maxCapabilities: 10})

	if len(items) != 1 || items[0].CapabilityID != "document.convert" {
		t.Fatalf("items = %#v, want merged document.convert only", items)
	}
	if got := items[0].SourceSurfaceIDs; len(got) != 2 || got[0] != "s1" || got[1] != "s2" {
		t.Fatalf("source ids = %#v, want s1 and s2", got)
	}
	if stage.Name != caltrace.ProposalStageCapability {
		t.Fatalf("stage.Name = %q, want capability", stage.Name)
	}
	if stage.Summary[caltrace.ProposalSummaryRaw] != 4 || stage.Summary[caltrace.ProposalSummarySelected] != 1 || stage.Summary[caltrace.ProposalSummaryKeep] != 2 || stage.Summary[caltrace.ProposalSummarySkip] != 2 {
		t.Fatalf("stage.Summary = %#v, want raw/selected/keep/skip counts", stage.Summary)
	}
}

func TestNormalizeCapabilityStageMarksCreatedReusedAndOutOfPolicy(t *testing.T) {
	policy := CapabilityPolicy{
		PreferredSubjects:   []string{"document"},
		PreferredOperations: []string{"convert"},
	}
	surfaces := []surface{{ID: "s1", Name: "convert"}, {ID: "s2", Name: "sign"}}
	req := Request{
		Catalog: []core.Capability{{ID: "document.convert"}},
	}
	input := []capabilityPlan{
		{CapabilityID: "document.convert", SourceSurfaceIDs: []string{"s1"}},
		{CapabilityID: "certificate.sign", SourceSurfaceIDs: []string{"s2"}},
	}

	items, stage := normalizeCapabilities(input, policy, surfaces, req, profile{maxCapabilities: 10})

	if len(items) != 2 {
		t.Fatalf("items = %#v, want two kept capabilities", items)
	}
	if stage.Summary[caltrace.ProposalSummaryReused] != 1 || stage.Summary[caltrace.ProposalSummaryCreated] != 1 || stage.Summary[caltrace.ProposalSummaryOutOfPolicy] != 1 {
		t.Fatalf("stage.Summary = %#v, want reused/created/out_of_policy counts", stage.Summary)
	}
}

func TestNormalizeCapabilityStageAppliesDebugFilter(t *testing.T) {
	policy := CapabilityPolicy{
		PreferredSubjects:   []string{"document", "text"},
		PreferredOperations: []string{"convert", "encode"},
	}
	surfaces := []surface{{ID: "s1", Name: "convert"}, {ID: "s2", Name: "encode"}}
	req := Request{DebugFilter: "text.encode"}
	input := []capabilityPlan{
		{CapabilityID: "document.convert", SourceSurfaceIDs: []string{"s1"}},
		{CapabilityID: "text.encode", SourceSurfaceIDs: []string{"s2"}},
	}

	items, stage := normalizeCapabilities(input, policy, surfaces, req, profile{maxCapabilities: 10})

	if len(items) != 1 || items[0].CapabilityID != "text.encode" {
		t.Fatalf("items = %#v, want debug-filtered text.encode", items)
	}
	if stage.Summary[caltrace.ProposalSummarySkip] != 1 || stage.Summary[caltrace.ProposalSummarySelected] != 1 {
		t.Fatalf("stage.Summary = %#v, want one skipped and one selected", stage.Summary)
	}
}

func TestExistingCapabilitiesFiltersOldDiscriminatorIDs(t *testing.T) {
	req := Request{
		Catalog: []core.Capability{
			{ID: "document.export_pdf"},
			{ID: "document.convert", Description: " Convert documents. "},
			{ID: "text.base64_encode"},
		},
	}

	items := existingCapabilities(req, 10)

	if len(items) != 1 || items[0].ID != "document.convert" || items[0].Description != "Convert documents." {
		t.Fatalf("items = %#v, want only valid existing capability with description", items)
	}
}
