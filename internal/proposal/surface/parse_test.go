package surface

import (
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/proposal/policy"
)

func TestParseKeepsAllowedSurfaceItems(t *testing.T) {
	raw := `{"surface_items":[{"id":"s1","kind":"command","name":"convert","usage":" convert --in <input> --out <output> ","decision":"keep"}]}`

	items, stage, err := Parse(raw, &Request{Policy: policy.Default().Surface})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Usage != "convert --in <input> --out <output>" {
		t.Fatalf("usage = %q, want trimmed usage", items[0].Usage)
	}
	if stage.Summary[model.ProposalSummaryKeep] != 1 {
		t.Fatalf("keep summary = %d, want 1", stage.Summary[model.ProposalSummaryKeep])
	}
}

func TestParseSkipsDuplicateSurfaceItems(t *testing.T) {
	raw := `{"surface_items":[{"id":"s1","kind":"command","name":"convert"},{"id":"s2","kind":"command","name":"convert"}]}`

	items, stage, err := Parse(raw, &Request{Policy: policy.Default().Surface})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if stage.Summary[model.ProposalSummarySkip] != 1 {
		t.Fatalf("skip summary = %d, want 1", stage.Summary[model.ProposalSummarySkip])
	}
}
