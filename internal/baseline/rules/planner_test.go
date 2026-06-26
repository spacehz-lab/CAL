package rules

import (
	"context"
	"os"
	"testing"

	"github.com/spacehz-lab/cal/internal/proposal"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestDeterministicProbePlannerPlansDocumentExportPDF(t *testing.T) {
	plan, err := planDeterministicProbe(t, "document.export_pdf")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if plan.Inputs["source"] == "" || plan.Inputs["target"] == "" {
		t.Fatalf("inputs = %#v, want source and target", plan.Inputs)
	}
	if _, err := os.Stat(plan.Inputs["source"].(string)); err != nil {
		t.Fatalf("source missing: %v", err)
	}
	if plan.Verifier.ID != verifierFileParsePDF {
		t.Fatalf("verifier = %#v, want PDF parse verifier", plan.Verifier)
	}
}

func TestDeterministicProbePlannerPlansImageResize(t *testing.T) {
	plan, err := planDeterministicProbe(t, "image.resize")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if plan.Inputs["source"] == "" || plan.Inputs["target"] == "" || plan.Inputs["width"] != 12 || plan.Inputs["height"] != 8 {
		t.Fatalf("inputs = %#v, want image resize probe inputs", plan.Inputs)
	}
	if _, err := os.Stat(plan.Inputs["source"].(string)); err != nil {
		t.Fatalf("source missing: %v", err)
	}
	if plan.Verifier.ID != verifierImageDimensions {
		t.Fatalf("verifier = %#v, want image dimension verifier", plan.Verifier)
	}
}

func TestDeterministicProbePlannerRejectsUnsupportedCapability(t *testing.T) {
	if _, err := planDeterministicProbe(t, "media.convert"); err == nil {
		t.Fatal("Plan() error = nil, want unsupported capability error")
	}
}

func TestDeterministicProbePlannerRequiresWorkDir(t *testing.T) {
	planner := NewProbePlanner()
	_, err := planner.Plan(context.Background(), proposal.ProbePlanRequest{
		Candidate: caltrace.Candidate{CapabilityID: "document.export_pdf"},
	})
	if err == nil {
		t.Fatal("Plan() error = nil, want missing work directory error")
	}
}

func planDeterministicProbe(t *testing.T, capabilityID string) (proposal.ProbePlan, error) {
	t.Helper()
	planner := NewProbePlanner()
	return planner.Plan(context.Background(), proposal.ProbePlanRequest{
		Candidate: caltrace.Candidate{CapabilityID: capabilityID},
		WorkDir:   t.TempDir(),
	})
}
