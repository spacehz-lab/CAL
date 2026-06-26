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

func TestSipsAcquisitionPromotesRealLocalCLIBinding(t *testing.T) {
	if os.Getenv("CAL_LOCAL_CLI_E2E") != "1" {
		t.Skip("set CAL_LOCAL_CLI_E2E=1 to run local real-CLI e2e")
	}
	if goruntime.GOOS != "darwin" {
		t.Skip("sips integration requires macOS")
	}
	providerPath := "/usr/bin/sips"
	if _, err := os.Stat(providerPath); err != nil {
		t.Skipf("sips not available: %v", err)
	}

	repo := e2etest.RepoRoot(t)
	temp := t.TempDir()
	calctlBin := filepath.Join(temp, "calctl")
	caldBin := filepath.Join(temp, "cald")
	e2etest.Build(t, repo, calctlBin, "./cmd/calctl")
	e2etest.Build(t, repo, caldBin, "./cmd/cald")

	home := filepath.Join(temp, "home")
	e2etest.WriteImageDimensionsVerifier(t, home, "image_dimensions_match")
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
		t.Fatalf("acquisition discovery = %#v, want one promoted sips binding", acquisition)
	}
	e2etest.AssertPromotionAction(t, home, acquisition.TraceID, "created", "created")

	trace := e2etest.ReadJSONFile[caltrace.Trace](t, filepath.Join(home, "discovery", acquisition.TraceID, "trace.json"))
	if len(trace.Candidates) != 1 || trace.Candidates[0].Source != "rules:cli_help_sips_resize" {
		t.Fatalf("trace candidates = %#v, want sips help candidate", trace.Candidates)
	}
	if !e2etest.HasObservation(trace.Observations, "help", "--resampleHeightWidth") {
		t.Fatalf("trace observations = %#v, want help observation describing resize", trace.Observations)
	}
	if len(trace.Probes) != 1 || !trace.Probes[0].Passed || trace.Probes[0].Verifier.ID != "image_dimensions_match" {
		t.Fatalf("trace probes = %#v, want passing image_dimensions_match probe", trace.Probes)
	}

	source := filepath.Join(temp, "source.png")
	target := filepath.Join(temp, "target.png")
	e2etest.WritePNG(t, source, 5, 5)
	var runSuccess struct {
		Status   string             `json:"status"`
		Verified bool               `json:"verified"`
		Evidence []core.EvidenceRef `json:"evidence"`
	}
	e2etest.RunJSON(t, repo, env, &runSuccess, calctlBin, "runs", "create", "--capability-id", "image.resize", "--inputs-json", `{"source":`+strconv.Quote(source)+`,"target":`+strconv.Quote(target)+`,"width":12,"height":8}`, "--verify", "--json")
	if runSuccess.Status != "succeeded" || !runSuccess.Verified {
		t.Fatalf("run success = %#v, want verified success", runSuccess)
	}
	if len(runSuccess.Evidence) != 1 || runSuccess.Evidence[0].ID != "image_dimensions_match" {
		t.Fatalf("run evidence = %#v, want image_dimensions_match evidence", runSuccess.Evidence)
	}
	e2etest.AssertPNGDimensions(t, target, 12, 8)

	useTarget := filepath.Join(temp, "use-target.png")
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
	e2etest.RunJSON(t, repo, env, &useSuccess, calctlBin, "use", "--intent", "resize this image", "--inputs-json", `{"source":`+strconv.Quote(source)+`,"target":`+strconv.Quote(useTarget)+`,"width":16,"height":9}`, "--verify", "--json")
	if useSuccess.Status != "succeeded" || useSuccess.Selection.CapabilityID != "image.resize" || useSuccess.Selection.BindingID == "" || useSuccess.Selection.ProviderID != acquisition.Providers[0].ID {
		t.Fatalf("use success = %#v, want selected sips image.resize binding", useSuccess)
	}
	if useSuccess.Run.Status != "succeeded" || !useSuccess.Run.Verified || useSuccess.Run.BindingID != useSuccess.Selection.BindingID || useSuccess.Run.ProviderID != acquisition.Providers[0].ID {
		t.Fatalf("use run = %#v, want verified selected sips binding", useSuccess.Run)
	}
	e2etest.AssertPNGDimensions(t, useTarget, 16, 9)

	var refresh struct {
		State                string `json:"state"`
		TraceID              string `json:"trace_id"`
		CapabilitiesPromoted int    `json:"capabilities_promoted"`
		BindingsPromoted     int    `json:"bindings_promoted"`
	}
	e2etest.RunJSON(t, repo, env, &refresh, calctlBin, "discovery", "run", "--provider-id", acquisition.Providers[0].ID, "--mode", "rules", "--json")
	if refresh.State != "succeeded" || refresh.CapabilitiesPromoted != 0 || refresh.BindingsPromoted != 1 || refresh.TraceID == "" {
		t.Fatalf("refresh acquisition = %#v, want reused capability and refreshed binding", refresh)
	}
	e2etest.AssertPromotionAction(t, home, refresh.TraceID, "reused", "updated")

	var metrics e2etest.EvalMetricsOutput
	e2etest.RunJSON(t, repo, env, &metrics, calctlBin, "eval", "--json")
	if metrics.Summary.Providers != 1 || metrics.Summary.Capabilities != 1 || metrics.Summary.Bindings != 1 || metrics.Summary.PromotedBindings != 1 || metrics.Summary.Traces != 2 || metrics.Summary.Runs != 2 {
		t.Fatalf("eval summary = %#v, want sips closed-loop records", metrics.Summary)
	}
	if metrics.Acquisition.AttemptCount != 2 || metrics.Acquisition.CompletedCount != 2 || metrics.Acquisition.PromotionCount != 2 {
		t.Fatalf("eval acquisition counts = %#v, want two successful sips acquisitions", metrics.Acquisition)
	}
	if metrics.Acquisition.CapabilityCreatedCount != 1 || metrics.Acquisition.CapabilityReusedCount != 1 || metrics.Acquisition.BindingCreatedCount != 1 || metrics.Acquisition.BindingUpdatedCount != 1 {
		t.Fatalf("eval promotion actions = %#v, want created/reused and created/updated evidence", metrics.Acquisition)
	}
	if metrics.Acquisition.BindingPromotionRate != 1 || metrics.Acquisition.ProbeSuccessRate != 1 {
		t.Fatalf("eval acquisition rates = %#v, want successful sips acquisition rates", metrics.Acquisition)
	}
	if metrics.Reuse.RunCount != 2 || metrics.Reuse.RunSuccessCount != 2 || metrics.Reuse.VerifiedRunCount != 2 || metrics.Reuse.VerifiedSuccessRate != 1 || metrics.Reuse.VerifierFailureRate != 0 {
		t.Fatalf("eval reuse = %#v, want verified sips reuse", metrics.Reuse)
	}
}
