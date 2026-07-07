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

func candidate(capabilityID string, providerID string, args []any) model.Candidate {
	return model.Candidate{
		ProviderID:   providerID,
		CapabilityID: capabilityID,
		Execution:    model.Execution{Kind: model.ExecutionKindCLI, Spec: map[string]any{model.ExecutionSpecArgs: args}},
	}
}
