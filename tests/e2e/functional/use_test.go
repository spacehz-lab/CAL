package e2e

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	e2etest "github.com/spacehz-lab/cal/internal/testsupport/e2e"
)

func TestUseSelectsProviderScopedBinding(t *testing.T) {
	repo, calctlBin, caldBin := functionalBinaries(t)
	temp := t.TempDir()

	home := filepath.Join(temp, "home")
	firstPath := filepath.Join(temp, "first-exporter")
	secondPath := filepath.Join(temp, "second-exporter")
	e2etest.WriteFakeExporter(t, firstPath, e2etest.WriteParseablePDFCommand())
	e2etest.WriteFakeExporter(t, secondPath, e2etest.WriteParseablePDFCommand())

	env := e2etest.WithHomeEnv(os.Environ(), home)
	e2etest.StartCald(t, repo, env, caldBin)

	var first struct {
		State                string                    `json:"state"`
		CapabilitiesPromoted int                       `json:"capabilities_promoted"`
		BindingsPromoted     int                       `json:"bindings_promoted"`
		Providers            []e2etest.ProviderSummary `json:"providers"`
	}
	runDiscoveryForProviderPath(t, repo, env, calctlBin, firstPath, &first, "--mode", "rules", "--json")
	if first.State != "succeeded" || first.CapabilitiesPromoted != 1 || first.BindingsPromoted != 1 || len(first.Providers) != 1 {
		t.Fatalf("first acquisition = %#v, want created capability and binding", first)
	}

	var second struct {
		State                string                    `json:"state"`
		CapabilitiesPromoted int                       `json:"capabilities_promoted"`
		BindingsPromoted     int                       `json:"bindings_promoted"`
		Providers            []e2etest.ProviderSummary `json:"providers"`
	}
	runDiscoveryForProviderPath(t, repo, env, calctlBin, secondPath, &second, "--mode", "rules", "--json")
	if second.State != "succeeded" || second.CapabilitiesPromoted != 0 || second.BindingsPromoted != 1 || len(second.Providers) != 1 {
		t.Fatalf("second acquisition = %#v, want reused capability and second binding", second)
	}

	source := filepath.Join(temp, "source.txt")
	if err := os.WriteFile(source, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	var useSuccess struct {
		Status    string `json:"status"`
		Selection struct {
			CapabilityID string `json:"capability_id"`
			BindingID    string `json:"binding_id"`
			ProviderID   string `json:"provider_id"`
		} `json:"selection"`
		Run struct {
			Status     string         `json:"status"`
			Verified   bool           `json:"verified"`
			BindingID  string         `json:"binding_id"`
			ProviderID string         `json:"provider_id"`
			Inputs     map[string]any `json:"inputs"`
		} `json:"run"`
	}
	e2etest.RunJSON(t, repo, env, &useSuccess, calctlBin, "use", "--intent", "export this document as pdf", "--provider-id", second.Providers[0].ID, "--inputs-json", `{"source":`+strconv.Quote(source)+`}`, "--verify", "--json")
	if useSuccess.Status != "succeeded" || useSuccess.Selection.CapabilityID != "document.export_pdf" || useSuccess.Selection.ProviderID != second.Providers[0].ID {
		t.Fatalf("use success = %#v, want provider-scoped document.export_pdf selection", useSuccess)
	}
	if useSuccess.Run.Status != "succeeded" || !useSuccess.Run.Verified || useSuccess.Run.BindingID != useSuccess.Selection.BindingID || useSuccess.Run.ProviderID != second.Providers[0].ID {
		t.Fatalf("use run = %#v, want verified run on selected provider binding", useSuccess.Run)
	}
	target, ok := useSuccess.Run.Inputs["target"].(string)
	if !ok || target == "" {
		t.Fatalf("use run inputs = %#v, want generated target", useSuccess.Run.Inputs)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("use target missing: %v", err)
	}

	var metrics e2etest.EvalMetricsOutput
	e2etest.RunJSON(t, repo, env, &metrics, calctlBin, "eval", "--json")
	if metrics.Summary.Providers != 2 || metrics.Summary.Capabilities != 1 || metrics.Summary.Bindings != 2 || metrics.Summary.Runs != 1 {
		t.Fatalf("eval summary = %#v, want provider-scoped use records", metrics.Summary)
	}
}
