package proposal

import (
	"context"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestDraftEvidenceDropsFixtureOnlyLiteralCheck(t *testing.T) {
	client := &fakeStageClient{responses: [][]byte{[]byte(`{"verify":{"method":"execute","checks":[{"subject":{"type":"file","input":"target"},"predicate":"exists","params":{}},{"subject":{"type":"file","input":"target"},"predicate":"contains","params":{"value":"hello"}},{"subject":{"type":"exit_code"},"predicate":"equals","params":{"value":0}}]}}`)}}
	output, _, err := NewLLMProposer(client).draftEvidence(context.Background(), Request{
		Provider: core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI},
	}, 0, caltrace.Candidate{
		CapabilityID: "text.write",
		Description:  "Write text to a file.",
	}, probeMaterial{
		CandidateIndex: 0,
		Inputs:         map[string]any{"target": "{{workdir}}/out.txt"},
		Fixtures:       []Fixture{{Input: "source", Filename: "input.txt", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("draftEvidence() error = %v", err)
	}
	if output.Verify.Level != core.VerifyLevelL1 {
		t.Fatalf("verify level = %s, want L1 after dropping fixture-only semantic check", output.Verify.Level)
	}
	for _, check := range output.Verify.Checks {
		if check.Predicate == core.VerifyPredicateContains {
			t.Fatalf("verify checks = %#v, want fixture-only contains check removed", output.Verify.Checks)
		}
	}
}

func TestDraftEvidenceKeepsObservationBackedLiteralCheck(t *testing.T) {
	client := &fakeStageClient{responses: [][]byte{[]byte(`{"verify":{"method":"execute","checks":[{"subject":{"type":"file","input":"target"},"predicate":"contains","params":{"value":"CAL_PROBE_OK"}},{"subject":{"type":"exit_code"},"predicate":"equals","params":{"value":0}}]}}`)}}
	output, _, err := NewLLMProposer(client).draftEvidence(context.Background(), Request{
		Provider: core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI},
		Observations: []caltrace.Observation{{
			Type:    "cli_output",
			Content: map[string]any{"text": "A correct output must contain CAL_PROBE_OK."},
		}},
	}, 0, caltrace.Candidate{
		CapabilityID: "file.write",
		Description:  "Write a fixed marker.",
	}, probeMaterial{
		CandidateIndex: 0,
		Inputs:         map[string]any{"target": "{{workdir}}/out.txt"},
		Fixtures:       []Fixture{{Input: "source", Filename: "input.txt", Content: "CAL_PROBE_OK"}},
	})
	if err != nil {
		t.Fatalf("draftEvidence() error = %v", err)
	}
	if output.Verify.Level != core.VerifyLevelL3 {
		t.Fatalf("verify level = %s, want L3 for observation-backed literal", output.Verify.Level)
	}
	if len(output.Verify.Checks) != 2 {
		t.Fatalf("verify checks = %#v, want literal check retained", output.Verify.Checks)
	}
}

func TestDraftEvidenceFiltersFixtureOnlyContainsAnyValues(t *testing.T) {
	client := &fakeStageClient{responses: [][]byte{[]byte(`{"verify":{"method":"execute","checks":[{"subject":{"type":"file","input":"target"},"predicate":"contains_any","params":{"values":["hello","CAL_PROBE_OK"]}}]}}`)}}
	req := Request{
		Provider: core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI},
		Observations: []caltrace.Observation{{
			Type:    "cli_output",
			Content: map[string]any{"text": "A correct output must contain CAL_PROBE_OK."},
		}},
	}
	candidate := caltrace.Candidate{
		CapabilityID: "file.write",
		Description:  "Write a fixed marker.",
	}
	material := probeMaterial{
		CandidateIndex: 0,
		Inputs:         map[string]any{"target": "{{workdir}}/out.txt"},
		Fixtures:       []Fixture{{Input: "source", Filename: "input.txt", Content: "hello"}},
	}
	output, _, err := NewLLMProposer(client).draftEvidence(context.Background(), req, 0, candidate, material)
	if err != nil {
		t.Fatalf("draftEvidence() error = %v", err)
	}
	values, ok := output.Verify.Checks[0].Params["values"].([]string)
	if !ok || len(values) != 1 || values[0] != "CAL_PROBE_OK" {
		t.Fatalf("contains_any values = %#v, want only observation-backed value", output.Verify.Checks[0].Params["values"])
	}
}

func TestDraftEvidenceDerivesArtifactLevelFromChecks(t *testing.T) {
	client := &fakeStageClient{responses: [][]byte{[]byte(`{"verify":{"method":"execute","checks":[{"subject":{"type":"exit_code"},"predicate":"equals","params":{"value":0}},{"subject":{"type":"file","input":"target"},"predicate":"exists","params":{}},{"subject":{"type":"file","input":"target"},"predicate":"non_empty","params":{}}]}}`)}}
	output, _, err := NewLLMProposer(client).draftEvidence(context.Background(), Request{
		Provider: core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI},
	}, 0, caltrace.Candidate{
		CapabilityID: "file.write",
		Description:  "Write text to a file.",
	}, probeMaterial{
		CandidateIndex: 0,
		Inputs:         map[string]any{"target": "{{workdir}}/out.txt"},
	})
	if err != nil {
		t.Fatalf("draftEvidence() error = %v", err)
	}
	if output.Verify.Level != core.VerifyLevelL2 {
		t.Fatalf("verify level = %s, want L2 for basic artifact evidence", output.Verify.Level)
	}
}

func TestDraftEvidenceDerivesSemanticLevelFromChecks(t *testing.T) {
	client := &fakeStageClient{responses: [][]byte{[]byte(`{"verify":{"method":"execute","checks":[{"subject":{"type":"stdout"},"predicate":"hash_line_matches","params":{"source":"source","algorithm":"sha256"}}]}}`)}}
	output, _, err := NewLLMProposer(client).draftEvidence(context.Background(), Request{
		Provider: core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI},
	}, 0, caltrace.Candidate{
		CapabilityID: "file.checksum",
		Description:  "Compute a file checksum.",
	}, probeMaterial{
		CandidateIndex: 0,
		Inputs:         map[string]any{"source": "{{workdir}}/source.txt"},
	})
	if err != nil {
		t.Fatalf("draftEvidence() error = %v", err)
	}
	if output.Verify.Level != core.VerifyLevelL3 {
		t.Fatalf("verify level = %s, want L3 for semantic evidence", output.Verify.Level)
	}
}

func TestDraftEvidenceDerivesRegexAsArtifactLevel(t *testing.T) {
	client := &fakeStageClient{responses: [][]byte{[]byte(`{"verify":{"method":"execute","checks":[{"subject":{"type":"file","input":"target"},"predicate":"regex","params":{"pattern":"status"}}]}}`)}}
	output, _, err := NewLLMProposer(client).draftEvidence(context.Background(), Request{
		Provider: core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI},
	}, 0, caltrace.Candidate{
		CapabilityID: "json.inspect",
		Description:  "Inspect JSON output.",
	}, probeMaterial{
		CandidateIndex: 0,
		Inputs:         map[string]any{"target": "{{workdir}}/out.json"},
	})
	if err != nil {
		t.Fatalf("draftEvidence() error = %v", err)
	}
	if output.Verify.Level != core.VerifyLevelL2 {
		t.Fatalf("verify level = %s, want L2 for regex evidence", output.Verify.Level)
	}
}

func TestDraftEvidenceDefaultsContractToL1(t *testing.T) {
	client := &fakeStageClient{responses: [][]byte{[]byte(`{"verify":{"method":"contract","checks":[{"subject":{"type":"file","input":"missing"},"predicate":"exists","params":{}}]}}`)}}
	output, _, err := NewLLMProposer(client).draftEvidence(context.Background(), Request{
		Provider: core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI},
	}, 0, caltrace.Candidate{
		CapabilityID: "package.install",
		Description:  "Install a package.",
	}, probeMaterial{
		CandidateIndex: 0,
	})
	if err != nil {
		t.Fatalf("draftEvidence() error = %v", err)
	}
	if output.Verify.Level != core.VerifyLevelL1 || output.Verify.Method != core.VerifyMethodContract || len(output.Verify.Checks) != 1 {
		t.Fatalf("verify = %#v, want contract L1 with advisory checks retained", output.Verify)
	}
}
