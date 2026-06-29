package llm

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
	sharedllm "github.com/spacehz-lab/cal/internal/llm"
	"github.com/spacehz-lab/cal/internal/proposal"
	"github.com/spacehz-lab/cal/internal/runtime"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

var _ proposal.Proposer = (*Proposer)(nil)
var _ proposal.ProbePlanner = (*Proposer)(nil)

func TestProposerProposesCandidateFromLLMProposal(t *testing.T) {
	client := &fakeClient{content: []byte(llmProposalJSON()), model: "adapter-model"}
	proposer := NewProposer(client)

	response, err := proposer.Propose(context.Background(), proposal.Request{
		Provider: core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI, Path: "/bin/make-pdf"},
		Observations: []caltrace.Observation{{
			ProviderID: "provider_cli",
			Type:       "help",
			Source:     "/bin/make-pdf --help",
			Content:    map[string]any{"text": "make-pdf --in <file> --out <pdf>"},
		}},
		ExistingCapabilityIDs: []string{"document.export_pdf"},
	})
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}
	if len(response.Candidates) != 1 {
		t.Fatalf("candidates = %#v, want one", response.Candidates)
	}
	candidate := response.Candidates[0]
	if candidate.ProviderID != "provider_cli" || candidate.CapabilityID != "document.export_pdf" || candidate.Source != "proposal:llm" {
		t.Fatalf("candidate = %#v, want llm proposal candidate", candidate)
	}
	if candidate.Description == "" {
		t.Fatalf("candidate = %#v, want description", candidate)
	}
	if candidate.InputConstraints["target"] == nil {
		t.Fatalf("input constraints = %#v, want target constraint", candidate.InputConstraints)
	}
	if candidate.Provenance == nil || candidate.Provenance.Source != "proposal:llm" || candidate.Provenance.PromptVersion != "prompt-v1" || candidate.Provenance.Model != "adapter-model" || candidate.Provenance.SchemaVersion != "proposal.v1" || len(candidate.Provenance.ProposalHash) != 64 {
		t.Fatalf("provenance = %#v, want llm proposal provenance", candidate.Provenance)
	}
	if len(client.prompts) != 1 {
		t.Fatalf("prompts = %#v, want one prompt", client.prompts)
	}
	if !strings.Contains(client.prompts[0].User, "provider_cli") {
		t.Fatalf("prompt user = %q, want provider context", client.prompts[0].User)
	}
}

func TestProposerDoesNotRequireVerifierCatalogDuringPropose(t *testing.T) {
	content := strings.Replace(llmProposalJSON(), `"verifier": {"id": "file_parse_pdf"}`, `"verifier": {"id": "file_exists"}`, 1)
	response, err := NewProposer(&fakeClient{content: []byte(content)}).Propose(context.Background(), proposal.Request{
		Provider: core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI},
	})
	if err != nil {
		t.Fatalf("Propose() error = %v, want proposal accepted without verifier catalog validation", err)
	}
	if len(response.Candidates) != 1 || response.Candidates[0].CapabilityID != "document.export_pdf" {
		t.Fatalf("candidates = %#v, want parsed proposal candidate", response.Candidates)
	}
}

func TestProposerPlansProbeFromLoadedProposal(t *testing.T) {
	t.Setenv("CAL_HOME", t.TempDir())
	proposer := NewProposer(&fakeClient{content: []byte(llmProposalJSON())})
	response, err := proposer.Propose(context.Background(), proposal.Request{
		Provider: core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI},
	})
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}
	workDir := t.TempDir()

	plan, err := proposer.Plan(context.Background(), proposal.ProbePlanRequest{
		Candidate: response.Candidates[0],
		WorkDir:   workDir,
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if plan.Inputs["target"] != filepath.Join(workDir, "output.pdf") {
		t.Fatalf("inputs = %#v, want rendered target", plan.Inputs)
	}
	source, ok := plan.Inputs["source"].(string)
	if !ok || !strings.HasPrefix(source, workDir) {
		t.Fatalf("source = %#v, want fixture inside workdir", plan.Inputs["source"])
	}
	if !strings.HasPrefix(plan.Verifier.ID, "verifier_file_parse_pdf_") {
		t.Fatalf("verifier = %#v, want rewritten verifier package id", plan.Verifier)
	}
}

func TestProposerPlansGeneratedVerifierFromLLMProposal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CAL_HOME", home)
	proposer := NewProposer(&fakeClient{content: []byte(llmGeneratedVerifierProposalJSON())})
	response, err := proposer.Propose(context.Background(), proposal.Request{
		Provider: core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI},
	})
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}

	plan, err := proposer.Plan(context.Background(), proposal.ProbePlanRequest{
		Candidate: response.Candidates[0],
		WorkDir:   t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	target := plan.Inputs["target"].(string)
	if err := os.WriteFile(target, []byte("probe ok"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	evidence, _, err := runtime.NewRegistry().Verify(context.Background(), plan.Verifier, plan.Inputs)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if len(evidence) != 1 || evidence[0].ID != plan.Verifier.ID {
		t.Fatalf("evidence = %#v, want generated verifier evidence", evidence)
	}
}

func TestProposerRequiresProposeBeforePlan(t *testing.T) {
	_, err := NewProposer(&fakeClient{content: []byte(llmProposalJSON())}).Plan(context.Background(), proposal.ProbePlanRequest{})
	if !errors.Is(err, ErrNoProposal) {
		t.Fatalf("Plan() error = %v, want ErrNoProposal", err)
	}
}

func TestProposerRequiresClient(t *testing.T) {
	_, err := NewProposer(nil).Propose(context.Background(), proposal.Request{})
	if !errors.Is(err, sharedllm.ErrNoClient) {
		t.Fatalf("Propose() error = %v, want ErrNoClient", err)
	}
}

func TestProposerReturnsClientError(t *testing.T) {
	clientErr := errors.New("client failed")

	_, err := NewProposer(&fakeClient{err: clientErr}).Propose(context.Background(), proposal.Request{})
	if !errors.Is(err, clientErr) {
		t.Fatalf("Propose() error = %v, want client error", err)
	}
}

func TestProposerReturnsProposalParseError(t *testing.T) {
	_, err := NewProposer(&fakeClient{content: []byte(`{"candidates":[]}`)}).Propose(context.Background(), proposal.Request{})
	if err == nil {
		t.Fatal("Propose() error = nil, want proposal parse error")
	}
}

func TestProposerLogsInvalidProposalDiagnostics(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelInfo})))
	t.Cleanup(func() {
		slog.SetDefault(previous)
	})

	_, err := NewProposer(&fakeClient{
		content: []byte(`{"candidates":[],"probe_plans":[{"candidate_index":0,"verifier":{"id":"file_exists"}}]}`),
		model:   "test-model",
	}).Propose(context.Background(), proposal.Request{
		Provider: core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI},
	})
	if err == nil {
		t.Fatal("Propose() error = nil, want proposal parse error")
	}
	text := logs.String()
	for _, want := range []string{
		"proposal llm failed",
		"stage=parse",
		"error=\"proposal must include at least one candidate\"",
		"response_bytes=",
		"proposal_candidate_count=0",
		"probe_plan_count=1",
		"verifier_package_count=0",
		"model=test-model",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("logs missing %q: %s", want, text)
		}
	}
}

type fakeClient struct {
	content []byte
	err     error
	model   string
	prompts []sharedllm.Prompt
}

func (client *fakeClient) Model() string {
	return client.model
}

func (client *fakeClient) Complete(_ context.Context, prompt sharedllm.Prompt) ([]byte, error) {
	client.prompts = append(client.prompts, prompt)
	if client.err != nil {
		return nil, client.err
	}
	return client.content, nil
}

func llmProposalJSON() string {
	return `{
			"metadata": {"source": "forged", "prompt_version": "forged-prompt", "model": "forged-model", "schema_version": "forged.schema"},
			"verifier_packages": [{
				"id": "file_parse_pdf",
				"description": "Passes when the target artifact starts with a PDF marker.",
				"verify_py": "import json\nimport sys\nrequest = json.load(sys.stdin)\nverifier_id = request['verifier']['id']\ntarget = request['inputs']['target']\nwith open(target, 'rb') as handle:\n    content = handle.read(4)\nprint(json.dumps({'passed': content == b'%PDF', 'evidence': [{'id': verifier_id, 'type': verifier_id}], 'outputs': {'target': target}}))\n"
			}],
			"candidates": [{
				"capability_id": "document.export_pdf",
				"source": "proposal:forged",
				"description": "Export a document to a PDF artifact.",
				"input_constraints": {
					"source": {"type": "string", "description": "input document path"},
				"target": {"type": "string", "description": "output PDF path"}
			},
			"execution": {
				"kind": "cli",
				"spec": {"args": ["make-pdf", "--in", "{{source}}", "--out", "{{target}}"]}
			}
		}],
		"probe_plans": [{
			"candidate_index": 0,
			"inputs": {"target": "{{workdir}}/output.pdf"},
			"fixtures": [{"input": "source", "filename": "input.txt", "content": "hello\n"}],
			"verifier": {"id": "file_parse_pdf"}
		}]
	}`
}

func llmGeneratedVerifierProposalJSON() string {
	return `{
		"metadata": {"source": "llm", "prompt_version": "prompt-v1", "model": "fake-llm", "schema_version": "proposal.v1"},
		"verifier_packages": [{
			"id": "contains_probe_text",
			"description": "Passes when the target file contains the probe text.",
			"verify_py": "import json\nimport sys\nrequest = json.load(sys.stdin)\nverifier_id = request['verifier']['id']\ntarget = request['inputs']['target']\nwith open(target, 'r', encoding='utf-8') as handle:\n    content = handle.read()\nprint(json.dumps({'passed': 'probe ok' in content, 'evidence': [{'id': verifier_id, 'type': verifier_id}], 'outputs': {'target': target}}))\n"
		}],
		"candidates": [{
			"capability_id": "custom.echo_marker",
			"description": "Write a marker text artifact.",
			"execution": {
				"kind": "cli",
				"spec": {"args": ["write-marker", "--out", "{{target}}"]}
			}
		}],
		"probe_plans": [{
			"candidate_index": 0,
			"inputs": {"target": "{{workdir}}/marker.txt"},
			"verifier": {"id": "contains_probe_text"}
		}]
	}`
}
