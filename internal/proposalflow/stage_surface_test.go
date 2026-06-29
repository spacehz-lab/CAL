package proposalflow

import (
	"testing"

	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestNormalizeSurfaceItemsAppliesDefaultsAndTrim(t *testing.T) {
	items, err := normalizeSurfaceItems([]surfaceItem{{
		ID:             " s1 ",
		Name:           " convert ",
		Description:    " Convert input. ",
		EvidenceSource: " help ",
	}}, DefaultPolicy().Surface, profile{})
	if err != nil {
		t.Fatalf("normalizeSurfaceItems() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %#v, want one item", items)
	}
	item := items[0]
	if item.ID != "s1" || item.Name != "convert" || item.Kind != "command" || item.Decision != caltrace.ProposalDecisionKeep || item.EvidenceSource != "help" {
		t.Fatalf("item = %#v, want normalized defaults", item)
	}
}

func TestNormalizeSurfaceItemsDropsSkippedDeferredAndDisallowedKinds(t *testing.T) {
	policy := DefaultPolicy()
	policy.Surface.AllowedKinds = []string{"command"}
	items, err := normalizeSurfaceItems([]surfaceItem{
		{ID: "s1", Kind: "command", Name: "convert", Decision: caltrace.ProposalDecisionKeep},
		{ID: "s2", Kind: "command", Name: "later", Decision: caltrace.ProposalDecisionDefer},
		{ID: "s3", Kind: "command", Name: "ignored", Decision: caltrace.ProposalDecisionSkip},
		{ID: "s4", Kind: "option", Name: "--encode", Decision: caltrace.ProposalDecisionKeep},
	}, policy.Surface, profile{})
	if err != nil {
		t.Fatalf("normalizeSurfaceItems() error = %v", err)
	}
	if len(items) != 1 || items[0].Name != "convert" {
		t.Fatalf("items = %#v, want only kept allowed command", items)
	}
}

func TestNormalizeSurfaceItemsSkipsNamesAndPatterns(t *testing.T) {
	policy := DefaultPolicy()
	policy.Surface.SkipNames = []string{"HELP"}
	policy.Surface.SkipPatterns = []string{"^debug-"}
	items, err := normalizeSurfaceItems([]surfaceItem{
		{ID: "s1", Kind: "command", Name: "help", Decision: caltrace.ProposalDecisionKeep},
		{ID: "s2", Kind: "command", Name: "debug-dump", Decision: caltrace.ProposalDecisionKeep},
		{ID: "s3", Kind: "command", Name: "convert", Decision: caltrace.ProposalDecisionKeep},
	}, policy.Surface, profile{})
	if err != nil {
		t.Fatalf("normalizeSurfaceItems() error = %v", err)
	}
	if len(items) != 1 || items[0].Name != "convert" {
		t.Fatalf("items = %#v, want only non-skipped surface", items)
	}
}

func TestNormalizeSurfaceItemsDeduplicatesByKindAndName(t *testing.T) {
	items, err := normalizeSurfaceItems([]surfaceItem{
		{ID: "s1", Kind: "command", Name: "convert", Decision: caltrace.ProposalDecisionKeep},
		{ID: "s2", Kind: "command", Name: "CONVERT", Decision: caltrace.ProposalDecisionKeep},
		{ID: "s3", Kind: "option", Name: "convert", Decision: caltrace.ProposalDecisionKeep},
	}, DefaultPolicy().Surface, profile{})
	if err != nil {
		t.Fatalf("normalizeSurfaceItems() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("items = %#v, want command and option variants only", items)
	}
	if items[0].ID != "s1" || items[1].ID != "s3" {
		t.Fatalf("items = %#v, want first command and option retained", items)
	}
}

func TestNormalizeSurfaceItemsAppliesMaxSurfaceItems(t *testing.T) {
	items, err := normalizeSurfaceItems([]surfaceItem{
		{ID: "s1", Kind: "command", Name: "one", Decision: caltrace.ProposalDecisionKeep},
		{ID: "s2", Kind: "command", Name: "two", Decision: caltrace.ProposalDecisionKeep},
	}, DefaultPolicy().Surface, profile{maxSurfaceItems: 1})
	if err != nil {
		t.Fatalf("normalizeSurfaceItems() error = %v", err)
	}
	if len(items) != 1 || items[0].Name != "one" {
		t.Fatalf("items = %#v, want first item only", items)
	}
}

func TestNormalizeSurfaceItemsAcceptsOptionSurface(t *testing.T) {
	items, err := normalizeSurfaceItems([]surfaceItem{{
		ID:       "s1",
		Kind:     "option",
		Name:     "--encode",
		Decision: caltrace.ProposalDecisionKeep,
	}}, DefaultPolicy().Surface, profile{})
	if err != nil {
		t.Fatalf("normalizeSurfaceItems() error = %v", err)
	}
	if len(items) != 1 || items[0].Kind != "option" {
		t.Fatalf("items = %#v, want option surface", items)
	}
}

func TestNormalizeSurfaceStageRecordsFinalDecisions(t *testing.T) {
	policy := DefaultPolicy()
	policy.Surface.SkipNames = []string{"help"}
	items, stage, err := normalizeSurfaceStage([]surfaceItem{
		{ID: "s1", Kind: "command", Name: "convert", Decision: caltrace.ProposalDecisionKeep},
		{ID: "s2", Kind: "command", Name: "server", Decision: caltrace.ProposalDecisionDefer},
		{ID: "s3", Kind: "command", Name: "help", Decision: caltrace.ProposalDecisionKeep},
		{ID: "s4", Kind: "command", Name: "ignored", Decision: caltrace.ProposalDecisionSkip},
	}, policy.Surface, profile{})
	if err != nil {
		t.Fatalf("normalizeSurfaceStage() error = %v", err)
	}
	if len(items) != 1 || items[0].Name != "convert" {
		t.Fatalf("items = %#v, want only convert selected for Stage2", items)
	}
	if stage.Name != caltrace.ProposalStageSurface || stage.Summary[caltrace.ProposalSummaryRaw] != 4 || stage.Summary[caltrace.ProposalSummarySelected] != 1 || stage.Summary[caltrace.ProposalSummaryKeep] != 1 || stage.Summary[caltrace.ProposalSummaryDefer] != 1 || stage.Summary[caltrace.ProposalSummarySkip] != 2 {
		t.Fatalf("stage = %#v, want final decision summary", stage)
	}
	if len(stage.Items) != 4 || stage.Items[2].Name != "help" || stage.Items[2].Decision != caltrace.ProposalDecisionSkip {
		t.Fatalf("stage items = %#v, want local policy skip recorded for help", stage.Items)
	}
}
