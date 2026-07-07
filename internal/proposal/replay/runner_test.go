package replay

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/proposal"
)

func TestRunnerReplaysProposalWithActiveProvider(t *testing.T) {
	path := writeReplayFile(t, `{
  "metadata": {"source":"replay","prompt_version":"test-v1","model":"fixture","schema_version":"proposal.v1"},
  "candidates": [{
    "provider_id": "provider_from_file",
    "capability_id": "document.convert",
    "description": "Export a document.",
    "execution": {"kind":"cli","spec":{"args":["make-pdf","--in","{{source}}","--out","{{target}}"]}}
  }],
  "probe_plans": [{
    "candidate_index": 0,
    "inputs": {"target": "{{workdir}}/output.pdf"},
    "fixtures": [{"input":"source","filename":"input.txt","content":"hello\n"}],
    "verify": {"level":"L2","method":"execute","checks":[{"subject":{"type":"file","input":"target"},"predicate":"format","params":{"format":"pdf"}}]}
  }]
}`)
	result, err := NewRunner(path).Run(context.Background(), request("provider_active", ""))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(result.Candidates) != 1 || len(result.ProbePlans) != 1 {
		t.Fatalf("result = %#v, want one candidate and probe plan", result)
	}
	candidate := result.Candidates[0]
	if candidate.ProviderID != "provider_active" || candidate.Source != sourceReplay {
		t.Fatalf("candidate = %#v, want active provider and replay source", candidate)
	}
	if candidate.Provenance == nil || candidate.Provenance.ProposalHash == "" || len(candidate.Provenance.ProposalHash) != 64 {
		t.Fatalf("candidate provenance = %#v, want replay hash", candidate.Provenance)
	}
	if result.ProbePlans[0].CandidateIndex != 0 {
		t.Fatalf("probe plan index = %d, want 0", result.ProbePlans[0].CandidateIndex)
	}
	if result.Diagnostics == nil || result.Diagnostics.Model != "fixture" || len(result.Diagnostics.Stages) != 1 {
		t.Fatalf("diagnostics = %#v, want replay diagnostics", result.Diagnostics)
	}
}

func TestRunnerDoesNotFilterByAcquisitionHint(t *testing.T) {
	path := writeReplayFile(t, `{
  "candidates": [{
    "capability_id": "document.convert",
    "description": "Export a document.",
    "execution": {"kind":"cli","spec":{"args":["make-pdf","{{target}}"]}}
  }],
  "probe_plans": [{
    "candidate_index": 0,
    "inputs": {"target": "{{workdir}}/output.pdf"},
    "verify": {"level":"L2","method":"execute","checks":[{"subject":{"type":"file","input":"target"},"predicate":"format","params":{"format":"pdf"}}]}
  }]
}`)
	result, err := NewRunner(path).Run(context.Background(), request("provider_active", "resize an image"))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(result.Candidates) != 1 || result.Candidates[0].CapabilityID != "document.convert" {
		t.Fatalf("candidates = %#v, want unfiltered replay candidate", result.Candidates)
	}
}

func TestRunnerRejectsInvalidProbePlanIndex(t *testing.T) {
	path := writeReplayFile(t, `{
  "candidates": [{
    "capability_id": "document.convert",
    "description": "Export a document.",
    "execution": {"kind":"cli","spec":{"args":["make-pdf","{{target}}"]}}
  }],
  "probe_plans": [{"candidate_index": 9, "verify": {"level":"L2","method":"execute","checks":[{"subject":{"type":"file","input":"target"},"predicate":"format","params":{"format":"pdf"}}]}}]
}`)
	_, err := NewRunner(path).Run(context.Background(), request("provider_active", ""))
	if err == nil {
		t.Fatal("Run() error = nil, want invalid probe plan index error")
	}
}

func request(providerID string, hint string) *proposal.Request {
	return &proposal.Request{
		Provider: &model.Provider{ID: providerID, Kind: model.ProviderKindCLI, Path: "/bin/tool"},
		Hint:     hint,
	}
}

func writeReplayFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "proposal.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write replay file: %v", err)
	}
	return path
}
