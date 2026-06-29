package proposalflow

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spacehz-lab/cal/internal/core"
	sharedllm "github.com/spacehz-lab/cal/internal/llm"
	"github.com/spacehz-lab/cal/internal/runtime"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestLLMProposerRunsFourStages(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CAL_HOME", home)
	client := &fakeStageClient{responses: [][]byte{
		[]byte(`{"surface_items":[{"id":"s1","kind":"command","name":"export-pdf","description":"Export text to PDF.","decision":"keep"}]}`),
		[]byte(`{"capabilities":[{"capability_id":"document.convert","description":"Convert a document between formats.","source_surface_ids":["s1"],"confidence":"high"}]}`),
		[]byte(`{"candidates":[{"capability_id":"document.convert","description":"Convert a document between formats.","execution":{"kind":"cli","spec":{"args":["export-pdf","--source","{{source}}","--target","{{target}}"]}}}],"probe_material":[{"candidate_index":0,"inputs":{"target":"{{workdir}}/out.pdf"},"fixtures":[{"input":"source","filename":"input.txt","content":"hello"}]}]}`),
		[]byte(`{"verifier_packages":[{"id":"pdf_magic","description":"Check PDF magic bytes.","verify_py":"import json\nrequest=json.load(open(0))\nprint(json.dumps({\"passed\": True, \"evidence\": [{\"id\": request[\"verifier\"][\"id\"], \"type\": request[\"verifier\"][\"id\"], \"content\": {\"target\": (request.get(\"inputs\") or {}).get(\"target\")}}], \"outputs\": {\"target\": (request.get(\"inputs\") or {}).get(\"target\")}}))\n"}],"verifier":{"id":"pdf_magic"}}`),
	}}

	result, err := NewLLMProposer(client).Propose(context.Background(), Request{
		Provider: core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI},
		Observations: []caltrace.Observation{{
			Type:    "cli_output",
			Source:  "help",
			Content: map[string]any{"text": "export-pdf --source <path> --target <path>"},
		}},
	})
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}
	if len(result.Candidates) != 1 || len(result.ProbePlans) != 1 {
		t.Fatalf("result = %#v, want one candidate and probe plan", result)
	}
	candidate := result.Candidates[0]
	if candidate.ProviderID != "provider_cli" || candidate.CapabilityID != "document.convert" || candidate.Provenance == nil || candidate.Provenance.Model != "fake-stage-model" {
		t.Fatalf("candidate = %#v, want normalized LLM candidate", candidate)
	}
	plan := result.ProbePlans[0]
	if plan.CandidateIndex != 0 || plan.Verifier.ID == "pdf_magic" || plan.Verifier.ID == "" {
		t.Fatalf("probe plan = %#v, want generated verifier id", plan)
	}
	if _, err := os.Stat(filepath.Join(home, "verifiers", plan.Verifier.ID, "verify.py")); err != nil {
		t.Fatalf("generated verifier missing: %v", err)
	}
	if !runtime.DefaultRegistry().Supports(plan.Verifier.ID) {
		t.Fatalf("registry does not support generated verifier %q", plan.Verifier.ID)
	}
	if result.Diagnostics == nil || result.Diagnostics.SchemaVersion != cliProposalSchema || len(result.Diagnostics.Stages) != 3 {
		t.Fatalf("diagnostics = %#v, want surface, capability, and binding stages", result.Diagnostics)
	}
	stage := result.Diagnostics.Stages[0]
	if stage.Name != caltrace.ProposalStageSurface || stage.Summary[caltrace.ProposalSummaryRaw] != 1 || stage.Summary[caltrace.ProposalSummarySelected] != 1 || len(stage.Items) != 1 || stage.Items[0].Name != "export-pdf" {
		t.Fatalf("surface diagnostics = %#v, want exported Stage1 item", stage)
	}
	capabilityStage := result.Diagnostics.Stages[1]
	if capabilityStage.Name != caltrace.ProposalStageCapability || capabilityStage.Summary[caltrace.ProposalSummaryRaw] != 1 || capabilityStage.Summary[caltrace.ProposalSummarySelected] != 1 {
		t.Fatalf("capability diagnostics = %#v, want selected Stage2 item", capabilityStage)
	}
	bindingStage := result.Diagnostics.Stages[2]
	if bindingStage.Name != caltrace.ProposalStageBinding || bindingStage.Summary[caltrace.ProposalSummaryRaw] != 1 || bindingStage.Summary[caltrace.ProposalSummarySelected] != 1 {
		t.Fatalf("binding diagnostics = %#v, want selected Stage3 item", bindingStage)
	}
}

func TestLLMProposerHashesEachCandidateEvidenceIndependently(t *testing.T) {
	surface := []byte(`{"surface_items":[{"id":"s1","kind":"command","name":"convert","description":"Convert documents.","decision":"keep"}]}`)
	capability := []byte(`{"capabilities":[{"capability_id":"document.convert","description":"Convert a document.","source_surface_ids":["s1"],"confidence":"high"}]}`)
	binding := []byte(`{"candidates":[{"capability_id":"document.convert","description":"Convert a document with mode A.","execution":{"kind":"cli","spec":{"args":["convert-a","{{source}}","{{target}}"]}}},{"capability_id":"document.convert","description":"Convert a document with mode B.","execution":{"kind":"cli","spec":{"args":["convert-b","{{source}}","{{target}}"]}}}],"probe_material":[{"candidate_index":0,"inputs":{"target":"{{workdir}}/a.out"},"fixtures":[{"input":"source","filename":"input.txt","content":"hello"}]},{"candidate_index":1,"inputs":{"target":"{{workdir}}/b.out"},"fixtures":[{"input":"source","filename":"input.txt","content":"hello"}]}]}`)
	evidenceA := []byte(`{"verifier":{"id":"file_exists"}}`)
	evidenceB := []byte(`{"verifier":{"id":"file_exists"} }`)
	client := &fakeStageClient{responses: [][]byte{surface, capability, binding, evidenceA, evidenceB}}

	result, err := NewLLMProposer(client).Propose(context.Background(), Request{
		Provider: core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI},
		Observations: []caltrace.Observation{{
			Type:    "cli_output",
			Source:  "help",
			Content: map[string]any{"text": "convert-a and convert-b"},
		}},
	})
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}
	if len(result.Candidates) != 2 {
		t.Fatalf("candidates = %#v, want two candidates", result.Candidates)
	}
	raw := append([]byte{}, surface...)
	raw = append(raw, capability...)
	raw = append(raw, binding...)
	expectedA := proposalHash(append(append([]byte{}, raw...), evidenceA...))
	expectedB := proposalHash(append(append([]byte{}, raw...), evidenceB...))
	combinedB := proposalHash(append(append(append([]byte{}, raw...), evidenceA...), evidenceB...))
	if result.Candidates[0].Provenance == nil || result.Candidates[0].Provenance.ProposalHash != expectedA {
		t.Fatalf("first candidate provenance = %#v, want hash %q", result.Candidates[0].Provenance, expectedA)
	}
	if result.Candidates[1].Provenance == nil || result.Candidates[1].Provenance.ProposalHash != expectedB {
		t.Fatalf("second candidate provenance = %#v, want hash %q", result.Candidates[1].Provenance, expectedB)
	}
	if result.Candidates[1].Provenance.ProposalHash == combinedB {
		t.Fatalf("second candidate hash includes previous candidate evidence")
	}
}

func TestLLMProposerReturnsErrorWhenAllCapabilityBindingsFail(t *testing.T) {
	client := allBindingsFailClient{}
	_, err := NewLLMProposer(&client).Propose(context.Background(), Request{
		Provider: core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI},
		Observations: []caltrace.Observation{{
			Type:    "cli_output",
			Content: map[string]any{"text": "export-pdf and encode commands"},
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "binding/evidence pipelines failed") {
		t.Fatalf("Propose() error = %v, want all binding pipelines failed error", err)
	}
}

func TestLLMProposerReturnsDiagnosticsWhenSurfaceHasNoKeptItems(t *testing.T) {
	client := &fakeStageClient{responses: [][]byte{
		[]byte(`{"surface_items":[{"id":"s1","kind":"command","name":"server","decision":"defer"},{"id":"s2","kind":"command","name":"help","decision":"skip"}]}`),
	}}

	result, err := NewLLMProposer(client).Propose(context.Background(), Request{
		Provider: core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI},
		Observations: []caltrace.Observation{{
			Type:    "cli_output",
			Content: map[string]any{"text": "server and help"},
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "surface stage returned no kept surface items") {
		t.Fatalf("Propose() error = %v, want no kept surface error", err)
	}
	if result.Diagnostics == nil || len(result.Diagnostics.Stages) != 1 {
		t.Fatalf("diagnostics = %#v, want surface diagnostics on error", result.Diagnostics)
	}
	stage := result.Diagnostics.Stages[0]
	if stage.Summary[caltrace.ProposalSummaryRaw] != 2 || stage.Summary[caltrace.ProposalSummaryDefer] != 1 || stage.Summary[caltrace.ProposalSummarySkip] != 1 || stage.Summary[caltrace.ProposalSummarySelected] != 0 {
		t.Fatalf("surface diagnostics = %#v, want deferred/skipped summary", stage)
	}
}

func TestLLMProposerReturnsCapabilityDiagnosticsWhenCapabilityHasNoKeptItems(t *testing.T) {
	client := &fakeStageClient{responses: [][]byte{
		[]byte(`{"surface_items":[{"id":"s1","kind":"command","name":"export-pdf","decision":"keep"}]}`),
		[]byte(`{"capabilities":[{"capability_id":"document.export_pdf","description":"Export PDF.","source_surface_ids":["s1"],"confidence":"high"}]}`),
	}}

	result, err := NewLLMProposer(client).Propose(context.Background(), Request{
		Provider: core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI},
		Observations: []caltrace.Observation{{
			Type:    "cli_output",
			Content: map[string]any{"text": "export-pdf"},
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "capability stage returned no capabilities") {
		t.Fatalf("Propose() error = %v, want no capabilities error", err)
	}
	if result.Diagnostics == nil || len(result.Diagnostics.Stages) != 2 {
		t.Fatalf("diagnostics = %#v, want surface and capability diagnostics on error", result.Diagnostics)
	}
	stage := result.Diagnostics.Stages[1]
	if stage.Name != caltrace.ProposalStageCapability || stage.Summary[caltrace.ProposalSummarySkip] != 1 || stage.Summary[caltrace.ProposalSummarySelected] != 0 {
		t.Fatalf("capability diagnostics = %#v, want skipped invalid capability", stage)
	}
}

func TestLLMProposerRejectsBindingCandidateWithoutProbeMaterial(t *testing.T) {
	client := &fakeStageClient{responses: [][]byte{
		[]byte(`{"surface_items":[{"id":"s1","kind":"command","name":"export-pdf","description":"Export text to PDF.","decision":"keep"}]}`),
		[]byte(`{"capabilities":[{"capability_id":"document.convert","description":"Convert a document.","source_surface_ids":["s1"],"confidence":"high"}]}`),
		[]byte(`{"candidates":[{"capability_id":"document.convert","description":"Convert a document.","execution":{"kind":"cli","spec":{"args":["export-pdf","{{source}}","{{target}}"]}}}],"probe_material":[]}`),
	}}

	_, err := NewLLMProposer(client).Propose(context.Background(), Request{
		Provider: core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI},
		Observations: []caltrace.Observation{{
			Type:    "cli_output",
			Content: map[string]any{"text": "export-pdf"},
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "binding stage returned no usable candidates") {
		t.Fatalf("Propose() error = %v, want unusable binding error", err)
	}
}

func TestLLMProposerDebugFilterSkipsOtherCapabilities(t *testing.T) {
	client := &fakeStageClient{responses: [][]byte{
		[]byte(`{"surface_items":[{"id":"s1","kind":"command","name":"export-pdf","description":"Export text to PDF.","decision":"keep"},{"id":"s2","kind":"command","name":"encode","description":"Encode text.","decision":"keep"}]}`),
		[]byte(`{"capabilities":[{"capability_id":"document.convert","description":"Convert a document.","source_surface_ids":["s1"],"confidence":"high"},{"capability_id":"text.encode","description":"Encode text.","source_surface_ids":["s2"],"confidence":"high"}]}`),
		[]byte(`{"candidates":[{"capability_id":"document.convert","description":"Convert a document.","execution":{"kind":"cli","spec":{"args":["export-pdf","{{source}}","{{target}}"]}}}],"probe_material":[{"candidate_index":0,"inputs":{"target":"{{workdir}}/out.pdf"},"fixtures":[{"input":"source","filename":"input.txt","content":"hello"}]}]}`),
		[]byte(`{"verifier":{"id":"file_exists"}}`),
	}}

	result, err := NewLLMProposer(client).Propose(context.Background(), Request{
		Provider:    core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI},
		DebugFilter: "document.convert",
		Observations: []caltrace.Observation{{
			Type:    "cli_output",
			Content: map[string]any{"text": "export-pdf and encode commands"},
		}},
	})
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}
	if len(result.Candidates) != 1 || result.Candidates[0].CapabilityID != "document.convert" {
		t.Fatalf("candidates = %#v, want only debug-filtered capability", result.Candidates)
	}
	if client.index != len(client.responses) {
		t.Fatalf("LLM calls = %d, want %d without binding skipped capability", client.index, len(client.responses))
	}
}

func TestFinalVerifierInstallsGeneratedVerifierWithStableID(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CAL_HOME", home)
	hash := "test_proposal_hash"
	localID := "unit_generated_check"
	verifier, err := finalVerifier(evidenceOutput{
		VerifierPackages: []runtime.GeneratedVerifierPackage{{
			ID:       localID,
			VerifyPY: "import json\nprint(json.dumps({\"passed\": True}))\n",
		}},
		Verifier: core.Verifier{ID: localID},
	}, hash, 2, 3)
	if err != nil {
		t.Fatalf("finalVerifier() error = %v", err)
	}
	expectedID := generatedVerifierID(hash, 2, 3, localID)
	if verifier.ID != expectedID {
		t.Fatalf("verifier.ID = %q, want %q", verifier.ID, expectedID)
	}
	if !runtime.DefaultRegistry().Supports(expectedID) {
		t.Fatalf("registry does not support generated verifier %q", expectedID)
	}
	if _, err := os.Stat(filepath.Join(home, "verifiers", expectedID, "verify.py")); err != nil {
		t.Fatalf("generated verifier missing: %v", err)
	}
}

func TestLLMProposerRequiresClient(t *testing.T) {
	_, err := NewLLMProposer(nil).Propose(context.Background(), Request{})
	if err != sharedllm.ErrNoClient {
		t.Fatalf("Propose() error = %v, want ErrNoClient", err)
	}
}

func TestLLMProposerKeepsSuccessfulCapabilityWhenAnotherPipelineFails(t *testing.T) {
	client := stagePromptClient{}
	result, err := NewLLMProposer(&client).Propose(context.Background(), Request{
		Provider: core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI},
		Observations: []caltrace.Observation{{
			Type:    "cli_output",
			Content: map[string]any{"text": "export-pdf and encode commands"},
		}},
	})
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}
	if len(result.Candidates) != 1 || len(result.ProbePlans) != 1 {
		t.Fatalf("result = %#v, want one successful capability", result)
	}
	if result.Candidates[0].CapabilityID != "document.convert" {
		t.Fatalf("candidate = %#v, want document.convert from successful pipeline", result.Candidates[0])
	}
	if result.ProbePlans[0].CandidateIndex != 0 {
		t.Fatalf("probe plan = %#v, want reindexed candidate", result.ProbePlans[0])
	}
}

func TestProposeBindingsTimesOutSlowBindingPipeline(t *testing.T) {
	proposer := NewLLMProposer(&blockingStageClient{})
	req := Request{Provider: core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI}}
	run := proposer.newBindingRun(req, profile{concurrency: 1, bindingTimeout: 10 * time.Millisecond}, nil, nil, newLogger("provider_cli"))
	_, _, err := run.run(context.Background(), []capabilityPlan{{CapabilityID: "text.encode", Description: "Encode text."}})
	if err == nil || !strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
		t.Fatalf("bindingRun.run() error = %v, want deadline exceeded", err)
	}
}

type fakeStageClient struct {
	responses [][]byte
	index     int
	mu        sync.Mutex
}

type blockingStageClient struct{}

func (client *blockingStageClient) Complete(ctx context.Context, _ sharedllm.Prompt) ([]byte, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func (client *fakeStageClient) Complete(context.Context, sharedllm.Prompt) ([]byte, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.index >= len(client.responses) {
		return nil, sharedllm.ErrEmptyResponse
	}
	response := client.responses[client.index]
	client.index++
	return response, nil
}

func (client *fakeStageClient) Model() string {
	return "fake-stage-model"
}

type stagePromptClient struct {
	mu sync.Mutex
}

func (client *stagePromptClient) Complete(_ context.Context, prompt sharedllm.Prompt) ([]byte, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	switch {
	case prompt.System == cliSurfaceSystemPrompt:
		return []byte(`{"surface_items":[{"id":"s1","name":"export-pdf","decision":"keep"},{"id":"s2","name":"encode","decision":"keep"}]}`), nil
	case prompt.System == cliCapabilitySystemPrompt:
		return []byte(`{"capabilities":[{"capability_id":"document.convert","description":"Convert a document.","source_surface_ids":["s1"]},{"capability_id":"text.encode","description":"Encode text.","source_surface_ids":["s2"]}]}`), nil
	case prompt.System == cliBindingSystemPrompt && strings.Contains(prompt.User, `"capability_id":"document.convert"`):
		return []byte(`{"candidates":[{"capability_id":"document.convert","description":"Convert a document.","execution":{"kind":"cli","spec":{"args":["export-pdf","{{source}}","{{target}}"]}}}],"probe_material":[{"candidate_index":0,"inputs":{"target":"{{workdir}}/out.pdf"},"fixtures":[{"input":"source","filename":"input.txt","content":"hello"}]}]}`), nil
	case prompt.System == cliBindingSystemPrompt && strings.Contains(prompt.User, `"capability_id":"text.encode"`):
		return []byte(`{"candidates":[],"probe_material":[]}`), nil
	case prompt.System == cliEvidenceSystemPrompt:
		return []byte(`{"verifier":{"id":"file_exists"}}`), nil
	default:
		return nil, sharedllm.ErrEmptyResponse
	}
}

type allBindingsFailClient struct {
	mu sync.Mutex
}

func (client *allBindingsFailClient) Complete(_ context.Context, prompt sharedllm.Prompt) ([]byte, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	switch {
	case prompt.System == cliSurfaceSystemPrompt:
		return []byte(`{"surface_items":[{"id":"s1","name":"export-pdf","decision":"keep"},{"id":"s2","name":"encode","decision":"keep"}]}`), nil
	case prompt.System == cliCapabilitySystemPrompt:
		return []byte(`{"capabilities":[{"capability_id":"document.convert","description":"Convert a document.","source_surface_ids":["s1"]},{"capability_id":"text.encode","description":"Encode text.","source_surface_ids":["s2"]}]}`), nil
	case prompt.System == cliBindingSystemPrompt:
		return []byte(`{"candidates":[],"probe_material":[]}`), nil
	default:
		return nil, sharedllm.ErrEmptyResponse
	}
}
