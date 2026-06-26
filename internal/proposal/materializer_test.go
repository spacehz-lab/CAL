package proposal

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

func TestMaterializerProposesProposalCandidate(t *testing.T) {
	materializer := mustParse(t, proposalJSON("make-pdf", "proposal:test"))

	response, err := materializer.Propose(context.Background(), Request{
		Provider: core.Provider{ID: "provider_cli"},
	})
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}
	if len(response.Candidates) != 1 {
		t.Fatalf("candidates = %#v, want one", response.Candidates)
	}
	candidate := response.Candidates[0]
	if candidate.ProviderID != "provider_cli" || candidate.CapabilityID != "document.export_pdf" || candidate.Source != "proposal:test" {
		t.Fatalf("candidate = %#v, want proposal-backed candidate", candidate)
	}
	if candidate.Description == "" {
		t.Fatalf("candidate = %#v, want description", candidate)
	}
	if candidate.InputConstraints["target"] == nil {
		t.Fatalf("input constraints = %#v, want target constraint", candidate.InputConstraints)
	}
	if candidate.Provenance == nil || candidate.Provenance.Source != "proposal:test" || candidate.Provenance.PromptVersion != "prompt-v1" || candidate.Provenance.Model != "fixture-model" || candidate.Provenance.SchemaVersion != "proposal.v1" || len(candidate.Provenance.ProposalHash) != 64 {
		t.Fatalf("provenance = %#v, want replay provenance with proposal hash", candidate.Provenance)
	}
}

func TestParseWithMetadataOverridesProposalProvenance(t *testing.T) {
	materializer, err := ParseWithMetadata([]byte(proposalJSON("make-pdf", "proposal:test")), Metadata{
		Source:        "llm",
		PromptVersion: "prompt-v1",
		Model:         "adapter-model",
		SchemaVersion: "proposal.v1",
	})
	if err != nil {
		t.Fatalf("ParseWithMetadata() error = %v", err)
	}

	response, err := materializer.Propose(context.Background(), Request{
		Provider: core.Provider{ID: "provider_cli"},
	})
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}
	if len(response.Candidates) != 1 {
		t.Fatalf("candidates = %#v, want one", response.Candidates)
	}
	candidate := response.Candidates[0]
	if candidate.Source != "proposal:llm" {
		t.Fatalf("candidate source = %q, want adapter-owned source", candidate.Source)
	}
	if candidate.Provenance == nil || candidate.Provenance.Source != "proposal:llm" || candidate.Provenance.PromptVersion != "prompt-v1" || candidate.Provenance.Model != "adapter-model" || candidate.Provenance.SchemaVersion != "proposal.v1" {
		t.Fatalf("provenance = %#v, want adapter-owned provenance", candidate.Provenance)
	}
}

func TestParseReportsInvalidProposalStats(t *testing.T) {
	_, err := Parse([]byte(`{
		"verifier_packages": [{"id": "custom_check", "verify_py": "print('{}')\n"}],
		"candidates": [],
		"probe_plans": [{"candidate_index": 0, "verifier": {"id": "custom_check"}}]
	}`))
	var invalid InvalidProposalError
	if !errors.As(err, &invalid) {
		t.Fatalf("Parse() error = %v, want InvalidProposalError", err)
	}
	if invalid.Stats.CandidateCount != 0 || invalid.Stats.ProbePlanCount != 1 || invalid.Stats.VerifierPackageCount != 1 {
		t.Fatalf("stats = %#v, want proposal counts", invalid.Stats)
	}
	if len(invalid.ProposalHash) != 64 {
		t.Fatalf("proposal hash = %q, want sha256 hex", invalid.ProposalHash)
	}
}

func TestParseRejectsDuplicateVerifierPackageID(t *testing.T) {
	_, err := Parse([]byte(`{
		"verifier_packages": [
			{"id": "custom_check", "verify_py": "print('{}')\n"},
			{"id": "custom_check", "verify_py": "print('{}')\n"}
		],
		"candidates": [{
			"capability_id": "text.write_file",
			"description": "Write a text file artifact.",
			"execution": {"kind": "cli", "spec": {"args": ["write", "{{target}}"]}}
		}],
		"probe_plans": [{"candidate_index": 0, "verifier": {"id": "custom_check"}}]
	}`))
	var invalid InvalidProposalError
	if !errors.As(err, &invalid) || !strings.Contains(err.Error(), "duplicated") {
		t.Fatalf("Parse() error = %v, want duplicate verifier package error", err)
	}
}

func TestParseRejectsVerifierPackageIDReservedPrefix(t *testing.T) {
	_, err := Parse([]byte(`{
		"verifier_packages": [
			{"id": "verifier_custom_check", "verify_py": "print('{}')\n"}
		],
		"candidates": [{
			"capability_id": "text.write_file",
			"description": "Write a text file artifact.",
			"execution": {"kind": "cli", "spec": {"args": ["write", "{{target}}"]}}
		}],
		"probe_plans": [{"candidate_index": 0, "verifier": {"id": "verifier_custom_check"}}]
	}`))
	var invalid InvalidProposalError
	if !errors.As(err, &invalid) || !strings.Contains(err.Error(), "must not start with verifier_") {
		t.Fatalf("Parse() error = %v, want reserved verifier package prefix error", err)
	}
}

func TestMaterializerFiltersProviderAndHint(t *testing.T) {
	materializer := mustParse(t, proposalJSON("make-pdf", "proposal:test"))

	response, err := materializer.Propose(context.Background(), Request{
		Provider: core.Provider{ID: "provider_other"},
	})
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}
	if len(response.Candidates) != 0 {
		t.Fatalf("candidates = %#v, want provider mismatch filtered", response.Candidates)
	}

	response, err = materializer.Propose(context.Background(), Request{
		Provider: core.Provider{ID: "provider_cli"},
		Hint:     "image.resize",
	})
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}
	if len(response.Candidates) != 0 {
		t.Fatalf("candidates = %#v, want hint mismatch filtered", response.Candidates)
	}
}

func mustParse(t *testing.T, content string) Materializer {
	t.Helper()
	materializer, err := Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	return materializer
}

func proposalJSON(command, source string) string {
	return `{
		"metadata": {"source": "test", "prompt_version": "prompt-v1", "model": "fixture-model", "schema_version": "proposal.v1"},
		"candidates": [{
			"provider_id": "provider_cli",
			"capability_id": "document.export_pdf",
			"description": "Export a document to a PDF artifact.",
			"source": "` + source + `",
			"input_constraints": {
				"source": {"type": "string", "description": "input document path"},
				"target": {"type": "string", "description": "output PDF path"}
			},
			"execution": {
				"kind": "cli",
				"spec": {"args": ["` + command + `", "--in", "{{source}}", "--out", "{{target}}"]}
			},
			"rationale": "proposal maps the observed command to PDF export"
		}],
		"probe_plans": [{
			"candidate_index": 0,
			"inputs": {"target": "{{workdir}}/output.pdf"},
			"fixtures": [{"input": "source", "filename": "input.txt", "content": "hello\n"}],
			"verifier": {"id": "file_parse_pdf"},
			"rationale": "a parseable PDF proves the export result"
		}]
	}`
}
