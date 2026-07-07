package e2e

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestV1AcquisitionWithoutLLMReturnsStructuredError(t *testing.T) {
	repo, calctlBin, caldBin := functionalBinaries(t)
	temp := t.TempDir()
	home := filepath.Join(temp, "home")
	env := withHomeEnv(os.Environ(), home)
	startCald(t, repo, env, caldBin)

	var response errorResponse
	runFailJSON(t, repo, env, &response, calctlBin, "acquisition", "run", "--json")
	if response.Error.Code != "cald_unavailable" || response.Error.Message == "" {
		t.Fatalf("acquisition error = %#v, want structured cald_unavailable", response)
	}

	var missingProposal errorResponse
	runFailJSON(t, repo, env, &missingProposal, calctlBin, "acquisition", "run", "--mode", "replay", "--json")
	if missingProposal.Error.Code != "invalid_request" || missingProposal.Error.Message == "" {
		t.Fatalf("replay acquisition error = %#v, want structured invalid_request", missingProposal)
	}
}

func TestV1AcquisitionReplayPromotesRunnableBinding(t *testing.T) {
	repo, calctlBin, caldBin := functionalBinaries(t)
	temp := t.TempDir()
	home := filepath.Join(temp, "home")
	providerPath := filepath.Join(temp, "fake-exporter")
	writeFakeExecutable(t, providerPath)
	proposalPath := writeReplayProposal(t, filepath.Join(temp, "proposal.json"))

	env := withHomeEnv(os.Environ(), home)
	startCald(t, repo, env, caldBin)
	provider := addV1Provider(t, repo, env, calctlBin, providerPath)

	var acquisition acquisitionResponse
	runJSON(t, repo, env, &acquisition, calctlBin, "acquisition", "run", "--provider-id", provider.ID, "--hint", "convert a document to pdf", "--mode", "replay", "--proposal-path", proposalPath, "--json")
	assertAcquisitionPromoted(t, acquisition, "replay")
	assertDocumentConvertRun(t, repo, env, calctlBin, temp, provider.ID)
}

func TestV1AcquisitionRulesPromotesRunnableBinding(t *testing.T) {
	repo, calctlBin, caldBin := functionalBinaries(t)
	temp := t.TempDir()
	home := filepath.Join(temp, "home")
	providerPath := filepath.Join(temp, "fake-exporter")
	writeFakeExecutable(t, providerPath)

	env := withHomeEnv(os.Environ(), home)
	startCald(t, repo, env, caldBin)
	provider := addV1Provider(t, repo, env, calctlBin, providerPath)

	var acquisition acquisitionResponse
	runJSON(t, repo, env, &acquisition, calctlBin, "acquisition", "run", "--provider-id", provider.ID, "--hint", "convert a document to pdf", "--mode", "rules", "--json")
	assertAcquisitionPromoted(t, acquisition, "rules")
	assertDocumentConvertRun(t, repo, env, calctlBin, temp, provider.ID)
}

func assertAcquisitionPromoted(t *testing.T, response acquisitionResponse, mode string) {
	t.Helper()
	if response.TraceID == "" || response.CapabilitiesPromoted != 1 || response.BindingsPromoted != 1 || response.Trace == nil {
		t.Fatalf("%s acquisition = %#v, want one promoted binding and trace", mode, response)
	}
}

func assertDocumentConvertRun(t *testing.T, repo string, env []string, calctlBin string, temp string, providerID string) {
	t.Helper()
	source := filepath.Join(temp, "source.txt")
	target := filepath.Join(temp, "target.pdf")
	if err := os.WriteFile(source, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	var run runResponse
	runJSON(t, repo, env, &run, calctlBin, "runs", "create", "--capability-id", "document.convert", "--inputs-json", `{"source":`+strconv.Quote(source)+`,"target":`+strconv.Quote(target)+`}`, "--verify", "--json")
	if run.Run == nil || run.Run.Status != model.RunStatusSucceeded || !run.Run.Verified || run.Run.ProviderID != providerID {
		t.Fatalf("run = %#v, want verified document.convert run with provider %s", run, providerID)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("run target missing: %v", err)
	}
}

func writeReplayProposal(t *testing.T, path string) string {
	t.Helper()
	content := `{
  "metadata": {"source":"functional-replay","prompt_version":"test-v1","model":"fixture","schema_version":"proposal.v1"},
  "candidates": [{
    "capability_id": "document.convert",
    "description": "Export a document.",
    "execution": {"kind":"cli","spec":{"args":["export-pdf","--source","{{source}}","--target","{{target}}"]}}
  }],
  "probe_plans": [{
    "candidate_index": 0,
    "inputs": {"target": "{{workdir}}/output.pdf"},
    "fixtures": [{"input":"source","filename":"input.txt","content":"hello\n"}],
    "verify": {"level":"L2","method":"execute","checks":[{"subject":{"type":"file","input":"target"},"predicate":"format","params":{"format":"pdf"}}]}
  }]
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write replay proposal: %v", err)
	}
	return path
}
