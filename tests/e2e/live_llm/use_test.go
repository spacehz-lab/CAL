package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
	e2etest "github.com/spacehz-lab/cal/internal/testsupport/e2e"
)

func TestLiveLLMUseExtractsInputsAndFillsTarget(t *testing.T) {
	env := liveLLMEnv(t, filepath.Join(t.TempDir(), "home"))
	repo := e2etest.RepoRoot(t)
	temp := t.TempDir()
	calctlBin := filepath.Join(temp, "calctl")
	caldBin := filepath.Join(temp, "cald")
	e2etest.Build(t, repo, calctlBin, "./cmd/calctl")
	e2etest.Build(t, repo, caldBin, "./cmd/cald")
	e2etest.StartCald(t, repo, env, caldBin)

	providerPath := filepath.Join(temp, "proposal-exporter")
	e2etest.WriteProposalBackedExporter(t, providerPath)
	proposalPath := e2etest.WriteReplayProposal(t, filepath.Join(temp, "proposal.json"))
	var acquisition struct {
		State                string `json:"state"`
		CapabilitiesPromoted int    `json:"capabilities_promoted"`
		BindingsPromoted     int    `json:"bindings_promoted"`
	}
	runDiscoveryForProviderPath(t, repo, env, calctlBin, providerPath, &acquisition, "--proposal-path", proposalPath, "--json")
	if acquisition.State != "succeeded" || acquisition.CapabilitiesPromoted != 1 || acquisition.BindingsPromoted != 1 {
		t.Fatalf("acquisition = %#v, want replay-promoted capability", acquisition)
	}

	source := filepath.Join(temp, "source.txt")
	if err := os.WriteFile(source, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	intent := "export " + source + " as a PDF document"
	var useSuccess struct {
		Status    string `json:"status"`
		Selection struct {
			Source       string `json:"source"`
			CapabilityID string `json:"capability_id"`
			BindingID    string `json:"binding_id"`
		} `json:"selection"`
		Run struct {
			Status   string             `json:"status"`
			Verified bool               `json:"verified"`
			Inputs   map[string]any     `json:"inputs"`
			Evidence []core.EvidenceRef `json:"evidence"`
		} `json:"run"`
	}
	e2etest.RunJSON(t, repo, env, &useSuccess, calctlBin, "use", intent, "--verify", "--json")
	if useSuccess.Status != "succeeded" || useSuccess.Selection.Source != "llm" || useSuccess.Selection.CapabilityID != "document.export_pdf" || useSuccess.Selection.BindingID == "" {
		t.Fatalf("use success = %#v, want LLM-selected document.export_pdf", useSuccess)
	}
	if useSuccess.Run.Status != "succeeded" || !useSuccess.Run.Verified || len(useSuccess.Run.Evidence) != 1 {
		t.Fatalf("use run = %#v, want verified run", useSuccess.Run)
	}
	if useSuccess.Run.Inputs["source"] != source {
		t.Fatalf("run inputs = %#v, want source extracted from intent", useSuccess.Run.Inputs)
	}
	target, ok := useSuccess.Run.Inputs["target"].(string)
	if !ok || !strings.Contains(target, filepath.Join("cal", "artifacts")) {
		t.Fatalf("run inputs = %#v, want generated temporary target", useSuccess.Run.Inputs)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("generated target missing: %v", err)
	}
}
