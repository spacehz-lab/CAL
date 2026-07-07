package capability

import (
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/proposal/policy"
)

func TestParseKeepsValidCapabilityPlans(t *testing.T) {
	raw := `{"capabilities":[{"capability_id":"pdf.convert","source_surface_ids":["s1"],"confidence":"high"}]}`

	plans, stage, err := Parse(raw, &Request{
		Surfaces: []SurfaceItem{{ID: "s1"}},
		Policy:   policy.Default().Capability,
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("len(plans) = %d, want 1", len(plans))
	}
	if stage.Summary[model.ProposalSummaryCreated] != 1 {
		t.Fatalf("created summary = %d, want 1", stage.Summary[model.ProposalSummaryCreated])
	}
}

func TestParseSkipsUnknownSurfaceReferences(t *testing.T) {
	raw := `{"capabilities":[{"capability_id":"pdf.convert","source_surface_ids":["missing"]}]}`

	_, _, err := Parse(raw, &Request{
		Surfaces: []SurfaceItem{{ID: "s1"}},
		Policy:   policy.Default().Capability,
	})
	if err == nil {
		t.Fatal("Parse() error = nil, want error")
	}
}
