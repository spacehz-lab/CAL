package proposal

import (
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestSelectResultDedupesAndReindexesProbePlans(t *testing.T) {
	result, err := selectResult(&Result{
		Candidates: []model.Candidate{
			candidate("pdf.convert", "provider_one", []any{"tool", "convert"}),
			candidate("pdf.convert", "provider_one", []any{"tool", "convert"}),
			candidate("image.resize", "provider_one", []any{"tool", "resize"}),
		},
		ProbePlans: []ProbePlan{
			{CandidateIndex: 0},
			{CandidateIndex: 1},
			{CandidateIndex: 2},
		},
	}, selectOptions{ProviderID: "provider_one"})
	if err != nil {
		t.Fatalf("selectResult() error = %v", err)
	}
	if len(result.Candidates) != 2 {
		t.Fatalf("len(Candidates) = %d, want 2", len(result.Candidates))
	}
	if result.ProbePlans[0].CandidateIndex != 0 || result.ProbePlans[1].CandidateIndex != 1 {
		t.Fatalf("ProbePlan indexes = [%d %d], want [0 1]", result.ProbePlans[0].CandidateIndex, result.ProbePlans[1].CandidateIndex)
	}
}

func TestSelectResultRejectsMissingProbePlan(t *testing.T) {
	_, err := selectResult(&Result{
		Candidates: []model.Candidate{candidate("pdf.convert", "provider_one", []any{"tool", "convert"})},
	}, selectOptions{})
	if err == nil {
		t.Fatal("selectResult() error = nil, want error")
	}
}

func TestSelectResultKeepsOneCandidatePerCapability(t *testing.T) {
	result, err := selectResult(&Result{
		Candidates: []model.Candidate{
			candidate("text.write", "provider_one", []any{"write-note", "--in", "{{source}}", "--out", "{{target}}"}),
			candidate("text.write", "provider_one", []any{"write-note", "--out", "{{target}}"}),
			candidate("text.convert", "provider_one", []any{"make-pdf", "--in", "{{source}}", "--out", "{{target}}"}),
		},
		ProbePlans: []ProbePlan{
			{CandidateIndex: 0},
			{CandidateIndex: 1},
			{CandidateIndex: 2},
		},
	}, selectOptions{ProviderID: "provider_one"})
	if err != nil {
		t.Fatalf("selectResult() error = %v", err)
	}
	if len(result.Candidates) != 2 {
		t.Fatalf("len(Candidates) = %d, want 2", len(result.Candidates))
	}
	if result.Candidates[0].CapabilityID != "text.write" || result.Candidates[1].CapabilityID != "text.convert" {
		t.Fatalf("capability ids = [%s %s], want text.write/text.convert", result.Candidates[0].CapabilityID, result.Candidates[1].CapabilityID)
	}
	if result.ProbePlans[0].CandidateIndex != 0 || result.ProbePlans[1].CandidateIndex != 1 {
		t.Fatalf("ProbePlan indexes = [%d %d], want [0 1]", result.ProbePlans[0].CandidateIndex, result.ProbePlans[1].CandidateIndex)
	}
}

func candidate(capabilityID string, providerID string, args []any) model.Candidate {
	return model.Candidate{
		ProviderID:   providerID,
		CapabilityID: capabilityID,
		Execution:    model.Execution{Kind: model.ExecutionKindCLI, Spec: map[string]any{model.ExecutionSpecArgs: args}},
	}
}
