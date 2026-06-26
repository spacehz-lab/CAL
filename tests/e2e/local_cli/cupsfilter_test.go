package e2e

import (
	"os"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
	e2etest "github.com/spacehz-lab/cal/internal/testsupport/e2e"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestCupsfilterAcquisitionPromotesRealLocalCLIBinding(t *testing.T) {
	if os.Getenv("CAL_LOCAL_CLI_E2E") != "1" {
		t.Skip("set CAL_LOCAL_CLI_E2E=1 to run local real-CLI e2e")
	}
	if goruntime.GOOS != "darwin" {
		t.Skip("cupsfilter integration requires macOS")
	}
	providerPath := "/usr/sbin/cupsfilter"
	if _, err := os.Stat(providerPath); err != nil {
		t.Skipf("cupsfilter not available: %v", err)
	}

	repo := e2etest.RepoRoot(t)
	temp := t.TempDir()
	calctlBin := filepath.Join(temp, "calctl")
	caldBin := filepath.Join(temp, "cald")
	e2etest.Build(t, repo, calctlBin, "./cmd/calctl")
	e2etest.Build(t, repo, caldBin, "./cmd/cald")

	home := filepath.Join(temp, "home")
	e2etest.WritePDFMagicVerifier(t, home, "file_parse_pdf")
	env := e2etest.WithHomeEnv(os.Environ(), home)
	e2etest.StartCald(t, repo, env, caldBin)

	var acquisition struct {
		State                string                    `json:"state"`
		TraceID              string                    `json:"trace_id"`
		CapabilitiesPromoted int                       `json:"capabilities_promoted"`
		BindingsPromoted     int                       `json:"bindings_promoted"`
		Providers            []e2etest.ProviderSummary `json:"providers"`
	}
	e2etest.RunJSON(t, repo, env, &acquisition, calctlBin, "discovery", "run", "--provider-path", providerPath, "--mode", "rules", "--json")
	if acquisition.State != "succeeded" || acquisition.CapabilitiesPromoted != 1 || acquisition.BindingsPromoted != 1 || acquisition.TraceID == "" || len(acquisition.Providers) != 1 {
		t.Fatalf("acquisition discovery = %#v, want one promoted cupsfilter binding", acquisition)
	}
	provider := acquisition.Providers[0]

	trace := e2etest.ReadJSONFile[caltrace.Trace](t, filepath.Join(home, "discovery", acquisition.TraceID, "trace.json"))
	if len(trace.Candidates) != 1 || trace.Candidates[0].Source != "rules:cli_docs_cupsfilter_pdf" {
		t.Fatalf("trace candidates = %#v, want cupsfilter docs candidate", trace.Candidates)
	}
	if trace.Candidates[0].Execution.Spec[core.ExecutionSpecStdoutPathInput] != "target" {
		t.Fatalf("candidate execution = %#v, want stdout target execution", trace.Candidates[0].Execution)
	}
	if !e2etest.HasObservation(trace.Observations, "help", "cupsfilter") {
		t.Fatalf("trace observations = %#v, want help observation describing cupsfilter", trace.Observations)
	}
	if len(trace.Probes) != 1 || !trace.Probes[0].Passed || trace.Probes[0].Verifier.ID != "file_parse_pdf" {
		t.Fatalf("trace probes = %#v, want passing file_parse_pdf probe", trace.Probes)
	}

	source := filepath.Join(temp, "source.txt")
	target := filepath.Join(temp, "target.pdf")
	if err := os.WriteFile(source, []byte("CAL cupsfilter integration\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	var runSuccess struct {
		Status   string             `json:"status"`
		Verified bool               `json:"verified"`
		Evidence []core.EvidenceRef `json:"evidence"`
	}
	e2etest.RunJSON(t, repo, env, &runSuccess, calctlBin, "runs", "create", "--capability-id", "document.export_pdf", "--inputs-json", `{"source":`+strconv.Quote(source)+`,"target":`+strconv.Quote(target)+`}`, "--verify", "--json")
	if runSuccess.Status != "succeeded" || !runSuccess.Verified {
		t.Fatalf("run success = %#v, want verified success", runSuccess)
	}
	if len(runSuccess.Evidence) != 1 || runSuccess.Evidence[0].ID != "file_parse_pdf" {
		t.Fatalf("run evidence = %#v, want file_parse_pdf evidence", runSuccess.Evidence)
	}

	useTarget := filepath.Join(temp, "use-target.pdf")
	var useSuccess struct {
		Status    string `json:"status"`
		Selection struct {
			CapabilityID string `json:"capability_id"`
			BindingID    string `json:"binding_id"`
			ProviderID   string `json:"provider_id"`
		} `json:"selection"`
		Run struct {
			Status       string `json:"status"`
			Verified     bool   `json:"verified"`
			CapabilityID string `json:"capability_id"`
			BindingID    string `json:"binding_id"`
			ProviderID   string `json:"provider_id"`
		} `json:"run"`
	}
	e2etest.RunJSON(t, repo, env, &useSuccess, calctlBin, "use", "--intent", "export this text document as pdf", "--inputs-json", `{"source":`+strconv.Quote(source)+`,"target":`+strconv.Quote(useTarget)+`}`, "--verify", "--json")
	if useSuccess.Status != "succeeded" || useSuccess.Selection.CapabilityID != "document.export_pdf" || useSuccess.Selection.BindingID == "" || useSuccess.Selection.ProviderID != provider.ID {
		t.Fatalf("use success = %#v, want selected cupsfilter document.export_pdf binding", useSuccess)
	}
	if useSuccess.Run.Status != "succeeded" || !useSuccess.Run.Verified || useSuccess.Run.BindingID != useSuccess.Selection.BindingID || useSuccess.Run.ProviderID != provider.ID {
		t.Fatalf("use run = %#v, want verified selected cupsfilter binding", useSuccess.Run)
	}
	if _, err := os.Stat(useTarget); err != nil {
		t.Fatalf("use target missing: %v", err)
	}

	var metrics e2etest.EvalMetricsOutput
	e2etest.RunJSON(t, repo, env, &metrics, calctlBin, "eval", "--json")
	if metrics.Summary.Providers != 1 || metrics.Summary.Capabilities != 1 || metrics.Summary.Bindings != 1 || metrics.Summary.PromotedBindings != 1 || metrics.Summary.Traces != 1 || metrics.Summary.Runs != 2 {
		t.Fatalf("eval summary = %#v, want cupsfilter closed-loop records", metrics.Summary)
	}
	if metrics.Reuse.RunCount != 2 || metrics.Reuse.RunSuccessCount != 2 || metrics.Reuse.VerifiedRunCount != 2 || metrics.Reuse.VerifiedSuccessRate != 1 {
		t.Fatalf("eval reuse = %#v, want verified cupsfilter reuse", metrics.Reuse)
	}
}
