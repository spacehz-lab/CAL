package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/contract"
	"github.com/spacehz-lab/cal/internal/model"
)

func TestLiveLLMUseExtractsInputsAndFillsTarget(t *testing.T) {
	repo, calctlBin, caldBin := liveBinaries(t)
	workspace := liveLLMWorkspace(t)
	env := liveLLMEnv(t, workspace.home)
	startCald(t, repo, env, caldBin)

	providerPath := filepath.Join(workspace.temp, "proposal-exporter")
	writeLiveLLMExporter(t, providerPath)
	proposalPath := writeReplayProposal(t, filepath.Join(workspace.temp, "proposal.json"))
	_, acquisition := runAcquisitionForProviderPath(t, repo, env, calctlBin, providerPath, "--mode", "replay", "--proposal-path", proposalPath, "--json")
	assertAcquisitionCompleted(t, &acquisition, 1, 1)

	source := filepath.Join(workspace.temp, "source.txt")
	if err := os.WriteFile(source, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	intent := "export " + source + " as a PDF document"
	var useResponse contract.UseResponse
	runJSON(t, repo, env, &useResponse, calctlBin, "use", intent, "--verify", "--json")
	if useResponse.Status != model.RunStatusSucceeded || useResponse.Selection == nil || useResponse.Selection.Source != "llm" || useResponse.Selection.CapabilityID != "document.convert" || useResponse.Selection.BindingID == "" {
		t.Fatalf("use response = %#v, want LLM-selected document.convert", useResponse)
	}
	if useResponse.Run == nil || useResponse.Run.Status != model.RunStatusSucceeded || !useResponse.Run.Verified || len(useResponse.Run.Evidence) == 0 {
		t.Fatalf("use run = %#v, want verified run", useResponse.Run)
	}
	if useResponse.Run.Inputs["source"] != source {
		t.Fatalf("run inputs = %#v, want source extracted from intent", useResponse.Run.Inputs)
	}
	target, ok := useResponse.Run.Inputs["target"].(string)
	if !ok || !strings.Contains(target, filepath.Join("cal", "artifacts")) {
		t.Fatalf("run inputs = %#v, want generated temporary target", useResponse.Run.Inputs)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("generated target missing: %v", err)
	}
}
