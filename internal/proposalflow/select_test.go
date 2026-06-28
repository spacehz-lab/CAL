package proposalflow

import (
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestSelectFiltersDeduplicatesLimitsAndReindexes(t *testing.T) {
	result := Result{
		Candidates: []caltrace.Candidate{
			selectCandidate("provider_other", "file.checksum", "sha1"),
			selectCandidate("provider_cli", "file.checksum", "sha1"),
			selectCandidate("provider_cli", "file.checksum", "sha1"),
			selectCandidate("provider_cli", "text.encode", "encode"),
		},
		ProbePlans: []ProbePlan{
			{CandidateIndex: 0, Verifier: core.Verifier{ID: "file_exists"}},
			{CandidateIndex: 1, Verifier: core.Verifier{ID: "file_exists"}},
			{CandidateIndex: 2, Verifier: core.Verifier{ID: "file_exists"}},
			{CandidateIndex: 3, Verifier: core.Verifier{ID: "file_exists"}},
		},
	}

	selected, err := Select(result, SelectOptions{
		ProviderID: "provider_cli",
		Limit:      2,
	})
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if len(selected.Candidates) != 2 || len(selected.ProbePlans) != 2 {
		t.Fatalf("selected = %#v, want two candidates and probe plans", selected)
	}
	if selected.Candidates[0].CapabilityID != "file.checksum" || selected.Candidates[1].CapabilityID != "text.encode" {
		t.Fatalf("selected candidates = %#v, want deduplicated provider candidates", selected.Candidates)
	}
	for index, plan := range selected.ProbePlans {
		if plan.CandidateIndex != index {
			t.Fatalf("plan[%d].CandidateIndex = %d, want remapped index", index, plan.CandidateIndex)
		}
	}
}

func TestSelectRejectsMissingProbePlan(t *testing.T) {
	_, err := Select(Result{
		Candidates: []caltrace.Candidate{selectCandidate("provider_cli", "file.checksum", "sha1")},
	}, SelectOptions{ProviderID: "provider_cli"})
	if err == nil || !strings.Contains(err.Error(), "has no probe plan") {
		t.Fatalf("Select() error = %v, want missing probe plan error", err)
	}
}

func selectCandidate(providerID, capabilityID, command string) caltrace.Candidate {
	return caltrace.Candidate{
		ProviderID:   providerID,
		CapabilityID: capabilityID,
		Description:  "Test " + capabilityID + ".",
		Execution: core.Execution{
			Kind: core.ExecutionKindCLI,
			Spec: map[string]any{core.ExecutionSpecArgs: []string{command}},
		},
	}
}
