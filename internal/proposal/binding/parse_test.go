package binding

import (
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestParseKeepsCandidatesWithProbeMaterial(t *testing.T) {
	raw := `{"candidates":[{"execution":{"kind":"cli","spec":{"args":["tool"]}}}],"probe_material":[{"candidate_index":0,"inputs":{"sample":"value"}}]}`

	candidates, materials, stage, err := Parse(raw, &Request{
		Provider:   &model.Provider{ID: "provider_test"},
		Capability: Plan{CapabilityID: "pdf.convert", Description: "convert pdf"},
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(candidates) != 1 || len(materials) != 1 {
		t.Fatalf("Parse() lengths = %d candidates, %d materials; want 1, 1", len(candidates), len(materials))
	}
	if candidates[0].ProviderID != "provider_test" || candidates[0].CapabilityID != "pdf.convert" {
		t.Fatalf("candidate ids = %q/%q, want provider_test/pdf.convert", candidates[0].ProviderID, candidates[0].CapabilityID)
	}
	if stage.Summary[model.ProposalSummaryKeep] != 1 {
		t.Fatalf("keep summary = %d, want 1", stage.Summary[model.ProposalSummaryKeep])
	}
}

func TestParseNormalizesNilStdoutPathInput(t *testing.T) {
	raw := `{"candidates":[{"execution":{"kind":"cli","spec":{"args":["write","{{source}}","{{target}}"],"stdout_path_input":null}}}],"probe_material":[{"candidate_index":0,"inputs":{"source":"{{workdir}}/input.txt","target":"{{workdir}}/output.txt"}}]}`

	candidates, _, _, err := Parse(raw, &Request{
		Provider:   &model.Provider{ID: "provider_test"},
		Capability: Plan{CapabilityID: "file.write"},
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if _, ok := candidates[0].Execution.Spec[model.ExecutionSpecStdoutPathInput]; ok {
		t.Fatalf("stdout_path_input = %#v, want normalized away", candidates[0].Execution.Spec[model.ExecutionSpecStdoutPathInput])
	}
}

func TestParseRejectsCandidateWithoutProbeMaterial(t *testing.T) {
	raw := `{"candidates":[{"execution":{"kind":"cli","spec":{"args":["tool"]}}}]}`

	_, _, _, err := Parse(raw, &Request{Provider: &model.Provider{ID: "provider_test"}, Capability: Plan{CapabilityID: "pdf.convert"}})
	if err == nil {
		t.Fatal("Parse() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "without matching probe material") {
		t.Fatalf("Parse() error = %v, want missing probe material error", err)
	}
}

func TestParseRejectsStdoutPathInputWithoutProbeInput(t *testing.T) {
	raw := `{"candidates":[{"execution":{"kind":"cli","spec":{"args":["{{source}}"],"stdout_path_input":"target"}}}],"probe_material":[{"candidate_index":0,"inputs":{"source":"{{workdir}}/input.txt"}}]}`

	_, _, _, err := Parse(raw, &Request{Provider: &model.Provider{ID: "provider_test"}, Capability: Plan{CapabilityID: "file.checksum"}})
	if err == nil {
		t.Fatal("Parse() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "stdout_path_input missing probe input") {
		t.Fatalf("Parse() error = %v, want stdout path input error", err)
	}
}

func TestParseRejectsStdoutPathInputPointingToSource(t *testing.T) {
	raw := `{"candidates":[{"execution":{"kind":"cli","spec":{"args":["write-note","--out","{{target}}"],"stdout_path_input":"source"}}}],"probe_material":[{"candidate_index":0,"inputs":{"source":"{{workdir}}/input.txt","target":"{{workdir}}/output.txt"}}]}`

	_, _, _, err := Parse(raw, &Request{Provider: &model.Provider{ID: "provider_test"}, Capability: Plan{CapabilityID: "text.write"}})
	if err == nil {
		t.Fatal("Parse() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "stdout_path_input pointing to an input source") {
		t.Fatalf("Parse() error = %v, want invalid stdout path input error", err)
	}
}

func TestParseNormalizesPlaceholderProviderID(t *testing.T) {
	raw := `{"candidates":[{"provider_id":"optional","execution":{"kind":"cli","spec":{"args":["tool"]}}}],"probe_material":[{"candidate_index":0,"inputs":{}}]}`

	candidates, _, _, err := Parse(raw, &Request{
		Provider:   &model.Provider{ID: "provider_test"},
		Capability: Plan{CapabilityID: "text.convert"},
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if candidates[0].ProviderID != "provider_test" {
		t.Fatalf("provider id = %q, want provider_test", candidates[0].ProviderID)
	}
}
