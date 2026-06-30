package e2e

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
	e2etest "github.com/spacehz-lab/cal/internal/testsupport/e2e"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestLiveLLMAcquisitionPromotesNewCapability(t *testing.T) {
	workspace := liveLLMWorkspace(t)
	env := liveLLMEnv(t, workspace.home)
	repo := e2etest.RepoRoot(t)
	temp := workspace.temp
	calctlBin := filepath.Join(temp, "calctl")
	caldBin := filepath.Join(temp, "cald")
	e2etest.Build(t, repo, calctlBin, "./cmd/calctl")
	e2etest.Build(t, repo, caldBin, "./cmd/cald")
	e2etest.StartCald(t, repo, env, caldBin)

	providerPath := filepath.Join(temp, "live-exporter")
	writeLiveLLMExporter(t, providerPath)

	var acquisition struct {
		State                string `json:"state"`
		TraceID              string `json:"trace_id"`
		CapabilitiesPromoted int    `json:"capabilities_promoted"`
		BindingsPromoted     int    `json:"bindings_promoted"`
	}
	runDiscoveryForProviderPath(t, repo, env, calctlBin, providerPath, &acquisition, "--json")
	if acquisition.State != "succeeded" || acquisition.CapabilitiesPromoted != 1 || acquisition.BindingsPromoted != 1 || acquisition.TraceID == "" {
		t.Fatalf("acquisition = %#v, want new capability and binding promotion", acquisition)
	}

	home := e2etest.HomeFromEnv(env)
	trace := e2etest.ReadJSONFile[caltrace.Trace](t, filepath.Join(home, "discovery", acquisition.TraceID, "trace.json"))
	capabilityID := assertLiveLLMTrace(t, trace)
	capability := e2etest.ReadJSONFile[core.Capability](t, filepath.Join(home, "capabilities", capabilityID+".json"))
	if len(capability.Bindings) != 1 {
		t.Fatalf("capability = %#v, want one promoted binding", capability)
	}

	source := filepath.Join(temp, "source.txt")
	target := filepath.Join(temp, "target.pdf")
	if err := os.WriteFile(source, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	var runSuccess struct {
		Status   string             `json:"status"`
		Verified bool               `json:"verified"`
		Evidence []core.EvidenceRef `json:"evidence"`
	}
	e2etest.RunJSON(t, repo, env, &runSuccess, calctlBin, "runs", "create", "--capability-id", capabilityID, "--inputs-json", `{"source":`+strconv.Quote(source)+`,"target":`+strconv.Quote(target)+`}`, "--verify", "--json")
	if runSuccess.Status != "succeeded" || !runSuccess.Verified || len(runSuccess.Evidence) == 0 {
		t.Fatalf("run success = %#v, want verified live LLM capability reuse", runSuccess)
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
	e2etest.RunJSON(t, repo, env, &useSuccess, calctlBin, "use", "--intent", "convert a utf-8 text file into a valid pdf document", "--inputs-json", `{"source":`+strconv.Quote(source)+`,"target":`+strconv.Quote(useTarget)+`}`, "--verify", "--json")
	if useSuccess.Status != "succeeded" || useSuccess.Selection.CapabilityID != capabilityID || useSuccess.Selection.BindingID == "" {
		t.Fatalf("use success = %#v, want selected live LLM capability %s", useSuccess, capabilityID)
	}
	if useSuccess.Run.Status != "succeeded" || !useSuccess.Run.Verified || useSuccess.Run.BindingID != useSuccess.Selection.BindingID || useSuccess.Run.CapabilityID != capabilityID {
		t.Fatalf("use run = %#v, want verified selected live LLM binding", useSuccess.Run)
	}
	if _, err := os.Stat(useTarget); err != nil {
		t.Fatalf("use target missing: %v", err)
	}

	var metrics e2etest.EvalMetricsOutput
	e2etest.RunJSON(t, repo, env, &metrics, calctlBin, "eval", "--json")
	if metrics.Summary.Providers != 1 || metrics.Summary.Capabilities != 1 || metrics.Summary.Bindings != 1 || metrics.Summary.PromotedBindings != 1 || metrics.Summary.Traces != 1 || metrics.Summary.Runs != 2 {
		t.Fatalf("eval summary = %#v, want live LLM new capability closed-loop records", metrics.Summary)
	}
	if metrics.Reuse.RunCount != 2 || metrics.Reuse.RunSuccessCount != 2 || metrics.Reuse.VerifiedRunCount != 2 || metrics.Reuse.VerifiedSuccessRate != 1 {
		t.Fatalf("eval reuse = %#v, want verified live LLM new capability reuse", metrics.Reuse)
	}
}

func TestLiveLLMAcquisitionReusesExistingCapability(t *testing.T) {
	workspace := liveLLMWorkspace(t)
	env := liveLLMEnv(t, workspace.home)
	repo := e2etest.RepoRoot(t)
	temp := workspace.temp
	calctlBin := filepath.Join(temp, "calctl")
	caldBin := filepath.Join(temp, "cald")
	e2etest.Build(t, repo, calctlBin, "./cmd/calctl")
	e2etest.Build(t, repo, caldBin, "./cmd/cald")
	e2etest.StartCald(t, repo, env, caldBin)
	home := e2etest.HomeFromEnv(env)

	const seededCapabilityID = "text.convert"
	seedPath := filepath.Join(temp, "seed-exporter")
	e2etest.WriteProposalBackedExporter(t, seedPath)
	seedProposalPath := writeLiveLLMSeedConvertProposal(t, filepath.Join(temp, "seed-proposal.json"), seededCapabilityID)
	var seed struct {
		State                string `json:"state"`
		CapabilitiesPromoted int    `json:"capabilities_promoted"`
		BindingsPromoted     int    `json:"bindings_promoted"`
	}
	runDiscoveryForProviderPath(t, repo, env, calctlBin, seedPath, &seed, "--proposal-path", seedProposalPath, "--json")
	if seed.State != "succeeded" || seed.CapabilitiesPromoted != 1 || seed.BindingsPromoted != 1 {
		t.Fatalf("seed acquisition = %#v, want seeded %s", seed, seededCapabilityID)
	}

	providerPath := filepath.Join(temp, "live-exporter")
	writeLiveLLMExporter(t, providerPath)
	var acquisition struct {
		State                string                    `json:"state"`
		TraceID              string                    `json:"trace_id"`
		CapabilitiesPromoted int                       `json:"capabilities_promoted"`
		BindingsPromoted     int                       `json:"bindings_promoted"`
		Providers            []e2etest.ProviderSummary `json:"providers"`
	}
	runDiscoveryForProviderPath(t, repo, env, calctlBin, providerPath, &acquisition, "--json")
	if acquisition.State != "succeeded" || acquisition.CapabilitiesPromoted != 0 || acquisition.BindingsPromoted != 1 || acquisition.TraceID == "" || len(acquisition.Providers) != 1 {
		t.Fatalf("acquisition = %#v, want existing capability reuse with new binding", acquisition)
	}

	trace := e2etest.ReadJSONFile[caltrace.Trace](t, filepath.Join(home, "discovery", acquisition.TraceID, "trace.json"))
	assertLiveLLMTrace(t, trace)
	e2etest.AssertPromotionAction(t, home, acquisition.TraceID, "reused", "created")
	capability := e2etest.ReadJSONFile[core.Capability](t, filepath.Join(home, "capabilities", seededCapabilityID+".json"))
	if len(capability.Bindings) != 2 {
		t.Fatalf("capability = %#v, want seeded binding plus live LLM binding", capability)
	}

	source := filepath.Join(temp, "source.txt")
	target := filepath.Join(temp, "target.pdf")
	if err := os.WriteFile(source, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	var runSuccess struct {
		Status   string             `json:"status"`
		Verified bool               `json:"verified"`
		Evidence []core.EvidenceRef `json:"evidence"`
	}
	e2etest.RunJSON(t, repo, env, &runSuccess, calctlBin, "runs", "create", "--capability-id", seededCapabilityID, "--provider-id", acquisition.Providers[0].ID, "--inputs-json", `{"source":`+strconv.Quote(source)+`,"target":`+strconv.Quote(target)+`}`, "--verify", "--json")
	if runSuccess.Status != "succeeded" || !runSuccess.Verified {
		t.Fatalf("run success = %#v, want verified live LLM binding reuse", runSuccess)
	}
	if len(runSuccess.Evidence) == 0 {
		t.Fatalf("run evidence = %#v, want verification evidence", runSuccess.Evidence)
	}

	var metrics e2etest.EvalMetricsOutput
	e2etest.RunJSON(t, repo, env, &metrics, calctlBin, "eval", "--json")
	if metrics.Summary.Providers != 2 || metrics.Summary.Capabilities != 1 || metrics.Summary.Bindings != 2 || metrics.Summary.PromotedBindings != 2 || metrics.Summary.Traces != 2 || metrics.Summary.Runs != 1 {
		t.Fatalf("eval summary = %#v, want live LLM closed-loop records", metrics.Summary)
	}
	if metrics.Acquisition.AttemptCount != 2 || metrics.Acquisition.CompletedCount != 2 || metrics.Acquisition.PromotionCount != 2 {
		t.Fatalf("eval acquisition counts = %#v, want seed plus live LLM acquisition", metrics.Acquisition)
	}
	if metrics.Acquisition.CapabilityCreatedCount != 1 || metrics.Acquisition.CapabilityReusedCount != 1 || metrics.Acquisition.BindingCreatedCount != 2 || metrics.Acquisition.BindingUpdatedCount != 0 {
		t.Fatalf("eval promotion actions = %#v, want seed creation and live LLM capability reuse", metrics.Acquisition)
	}
	if metrics.Acquisition.BindingPromotionRate != 1 || metrics.Acquisition.ProbeSuccessRate != 1 {
		t.Fatalf("eval acquisition rates = %#v, want successful live LLM acquisition rates", metrics.Acquisition)
	}
	if metrics.Reuse.VerifiedSuccessRate != 1 || metrics.Reuse.VerifierFailureRate != 0 {
		t.Fatalf("eval reuse = %#v, want verified live LLM reuse", metrics.Reuse)
	}
}

func TestLiveLLMAcquisitionPromotesMultipleCapabilities(t *testing.T) {
	workspace := liveLLMWorkspace(t)
	env := liveLLMEnv(t, workspace.home)
	repo := e2etest.RepoRoot(t)
	temp := workspace.temp
	calctlBin := filepath.Join(temp, "calctl")
	caldBin := filepath.Join(temp, "cald")
	e2etest.Build(t, repo, calctlBin, "./cmd/calctl")
	e2etest.Build(t, repo, caldBin, "./cmd/cald")
	e2etest.StartCald(t, repo, env, caldBin)

	providerPath := filepath.Join(temp, "live-multi-tool")
	writeLiveLLMMultiCapabilityExporter(t, providerPath)

	var acquisition struct {
		State                string                    `json:"state"`
		TraceID              string                    `json:"trace_id"`
		CapabilitiesPromoted int                       `json:"capabilities_promoted"`
		BindingsPromoted     int                       `json:"bindings_promoted"`
		Providers            []e2etest.ProviderSummary `json:"providers"`
	}
	runDiscoveryForProviderPath(t, repo, env, calctlBin, providerPath, &acquisition, "--json")
	if acquisition.State != "succeeded" || acquisition.CapabilitiesPromoted != 2 || acquisition.BindingsPromoted != 2 || acquisition.TraceID == "" || len(acquisition.Providers) != 1 {
		t.Fatalf("acquisition = %#v, want two live LLM promoted capabilities", acquisition)
	}

	home := e2etest.HomeFromEnv(env)
	trace := e2etest.ReadJSONFile[caltrace.Trace](t, filepath.Join(home, "discovery", acquisition.TraceID, "trace.json"))
	pdfCapabilityID, noteCapabilityID := assertLiveLLMMultiCapTrace(t, trace)
	assertTraceVerifyForCapability(t, trace, pdfCapabilityID)
	assertTraceVerifyForCapability(t, trace, noteCapabilityID)
	for _, capabilityID := range []string{pdfCapabilityID, noteCapabilityID} {
		capability := e2etest.ReadJSONFile[core.Capability](t, filepath.Join(home, "capabilities", capabilityID+".json"))
		if len(capability.Bindings) != 1 {
			t.Fatalf("capability %s = %#v, want one promoted binding", capabilityID, capability)
		}
	}

	pdfSource := filepath.Join(temp, "source.txt")
	pdfTarget := filepath.Join(temp, "target.pdf")
	if err := os.WriteFile(pdfSource, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write pdf source: %v", err)
	}
	var pdfRun struct {
		Status   string             `json:"status"`
		Verified bool               `json:"verified"`
		Evidence []core.EvidenceRef `json:"evidence"`
	}
	e2etest.RunJSON(t, repo, env, &pdfRun, calctlBin, "runs", "create", "--capability-id", pdfCapabilityID, "--provider-id", acquisition.Providers[0].ID, "--inputs-json", `{"source":`+strconv.Quote(pdfSource)+`,"target":`+strconv.Quote(pdfTarget)+`}`, "--verify", "--json")
	if pdfRun.Status != "succeeded" || !pdfRun.Verified || len(pdfRun.Evidence) == 0 {
		t.Fatalf("pdf run = %#v, want verified %s reuse", pdfRun, pdfCapabilityID)
	}

	noteSource := filepath.Join(temp, "note-source.txt")
	noteTarget := filepath.Join(temp, "note.txt")
	if err := os.WriteFile(noteSource, []byte("note\n"), 0o644); err != nil {
		t.Fatalf("write note source: %v", err)
	}
	var noteRun struct {
		Status   string             `json:"status"`
		Verified bool               `json:"verified"`
		Evidence []core.EvidenceRef `json:"evidence"`
	}
	e2etest.RunJSON(t, repo, env, &noteRun, calctlBin, "runs", "create", "--capability-id", noteCapabilityID, "--provider-id", acquisition.Providers[0].ID, "--inputs-json", `{"source":`+strconv.Quote(noteSource)+`,"target":`+strconv.Quote(noteTarget)+`}`, "--verify", "--json")
	if noteRun.Status != "succeeded" || !noteRun.Verified || len(noteRun.Evidence) == 0 {
		t.Fatalf("note run = %#v, want verified %s reuse", noteRun, noteCapabilityID)
	}

	var metrics e2etest.EvalMetricsOutput
	e2etest.RunJSON(t, repo, env, &metrics, calctlBin, "eval", "--json")
	if metrics.Summary.Providers != 1 || metrics.Summary.Capabilities != 2 || metrics.Summary.Bindings != 2 || metrics.Summary.PromotedBindings != 2 || metrics.Summary.Traces != 1 || metrics.Summary.Runs != 2 {
		t.Fatalf("eval summary = %#v, want live LLM multi-cap closed-loop records", metrics.Summary)
	}
	if metrics.Acquisition.PromotionCount != 2 || metrics.Acquisition.CandidateCount != 2 || metrics.Acquisition.ProbePassCount != 2 {
		t.Fatalf("eval acquisition counts = %#v, want two live LLM promoted candidates", metrics.Acquisition)
	}
	if metrics.Acquisition.CapabilityCreatedCount != 2 || metrics.Acquisition.BindingCreatedCount != 2 || metrics.Acquisition.BindingPromotionRate != 1 || metrics.Acquisition.ProbeSuccessRate != 1 {
		t.Fatalf("eval acquisition rates = %#v, want successful live LLM multi-cap acquisition", metrics.Acquisition)
	}
	if metrics.Reuse.VerifiedSuccessRate != 1 || metrics.Reuse.VerifierFailureRate != 0 {
		t.Fatalf("eval reuse = %#v, want both promoted bindings reusable", metrics.Reuse)
	}
}

func TestLiveLLMAcquisitionGeneratesVerifySpec(t *testing.T) {
	workspace := liveLLMWorkspace(t)
	env := liveLLMEnv(t, workspace.home)
	repo := e2etest.RepoRoot(t)
	temp := workspace.temp
	calctlBin := filepath.Join(temp, "calctl")
	caldBin := filepath.Join(temp, "cald")
	e2etest.Build(t, repo, calctlBin, "./cmd/calctl")
	e2etest.Build(t, repo, caldBin, "./cmd/cald")
	e2etest.StartCald(t, repo, env, caldBin)

	providerPath := filepath.Join(temp, "live-marker-tool")
	writeLiveLLMMarkerWriter(t, providerPath)
	var acquisition struct {
		State                string                    `json:"state"`
		TraceID              string                    `json:"trace_id"`
		CapabilitiesPromoted int                       `json:"capabilities_promoted"`
		BindingsPromoted     int                       `json:"bindings_promoted"`
		Providers            []e2etest.ProviderSummary `json:"providers"`
	}
	runDiscoveryForProviderPath(t, repo, env, calctlBin, providerPath, &acquisition, "--json")
	if acquisition.State != "succeeded" || acquisition.CapabilitiesPromoted != 1 || acquisition.BindingsPromoted != 1 || acquisition.TraceID == "" || len(acquisition.Providers) != 1 {
		t.Fatalf("acquisition = %#v, want verify-spec live LLM acquisition", acquisition)
	}

	home := e2etest.HomeFromEnv(env)
	trace := e2etest.ReadJSONFile[caltrace.Trace](t, filepath.Join(home, "discovery", acquisition.TraceID, "trace.json"))
	if len(trace.Candidates) != 1 || len(trace.Probes) != 1 || !trace.Probes[0].Passed {
		t.Fatalf("trace = %#v, want one passed verify-spec probe", trace)
	}
	if core.VerifyLevelRank(trace.Probes[0].Verify.Level) < core.VerifyLevelRank(core.VerifyLevelL1) {
		t.Fatalf("trace probe = %#v, want L1+ verify", trace.Probes[0])
	}

	source := filepath.Join(temp, "source.txt")
	target := filepath.Join(temp, "marker.txt")
	if err := os.WriteFile(source, []byte("ignored\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	var runSuccess struct {
		Status   string             `json:"status"`
		Verified bool               `json:"verified"`
		Evidence []core.EvidenceRef `json:"evidence"`
	}
	e2etest.RunJSON(t, repo, env, &runSuccess, calctlBin, "runs", "create", "--capability-id", trace.Candidates[0].CapabilityID, "--provider-id", acquisition.Providers[0].ID, "--inputs-json", `{"source":`+strconv.Quote(source)+`,"target":`+strconv.Quote(target)+`}`, "--verify", "--json")
	if runSuccess.Status != "succeeded" || !runSuccess.Verified || len(runSuccess.Evidence) == 0 {
		t.Fatalf("run success = %#v, want verify-spec reuse", runSuccess)
	}

	var metrics e2etest.EvalMetricsOutput
	e2etest.RunJSON(t, repo, env, &metrics, calctlBin, "eval", "--json")
	if metrics.Summary.Providers != 1 || metrics.Summary.Capabilities != 1 || metrics.Summary.Bindings != 1 || metrics.Summary.PromotedBindings != 1 || metrics.Summary.Traces != 1 || metrics.Summary.Runs != 1 {
		t.Fatalf("eval summary = %#v, want verify-spec closed-loop records", metrics.Summary)
	}
	if metrics.Reuse.RunCount != 1 || metrics.Reuse.RunSuccessCount != 1 || metrics.Reuse.VerifiedRunCount != 1 || metrics.Reuse.VerifiedSuccessRate != 1 {
		t.Fatalf("eval reuse = %#v, want verify-spec reuse evidence", metrics.Reuse)
	}
}

func TestLiveLLMAcquisitionHandlesNestedCommandContractAndExecute(t *testing.T) {
	workspace := liveLLMWorkspace(t)
	env := liveLLMEnv(t, workspace.home)
	repo := e2etest.RepoRoot(t)
	temp := workspace.temp
	calctlBin := filepath.Join(temp, "calctl")
	caldBin := filepath.Join(temp, "cald")
	e2etest.Build(t, repo, calctlBin, "./cmd/calctl")
	e2etest.Build(t, repo, caldBin, "./cmd/cald")
	e2etest.StartCald(t, repo, env, caldBin)

	providerPath := filepath.Join(temp, "live-nested-pm")
	writeLiveLLMNestedPackageManager(t, providerPath)

	var contractAcquisition struct {
		State                string                    `json:"state"`
		TraceID              string                    `json:"trace_id"`
		CapabilitiesPromoted int                       `json:"capabilities_promoted"`
		BindingsPromoted     int                       `json:"bindings_promoted"`
		Providers            []e2etest.ProviderSummary `json:"providers"`
	}
	provider := runDiscoveryForProviderPath(t, repo, env, calctlBin, providerPath, &contractAcquisition, "--capability-id", "package.install", "--json")
	if contractAcquisition.State != "succeeded" || contractAcquisition.CapabilitiesPromoted != 1 || contractAcquisition.BindingsPromoted < 1 || contractAcquisition.TraceID == "" || len(contractAcquisition.Providers) != 1 {
		t.Fatalf("contract acquisition = %#v, want package.install contract binding", contractAcquisition)
	}

	home := e2etest.HomeFromEnv(env)
	contractTrace := e2etest.ReadJSONFile[caltrace.Trace](t, filepath.Join(home, "discovery", contractAcquisition.TraceID, "trace.json"))
	assertLiveLLMCapabilityProbe(t, contractTrace, "package.install", core.VerifyLevelL1, core.VerifyMethodContract)

	var contractRun struct {
		Status string `json:"status"`
		Error  struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	e2etest.RunFailJSON(t, repo, env, &contractRun, calctlBin, "runs", "create", "--capability-id", "package.install", "--provider-id", provider.ID, "--inputs-json", `{"package":"cal-contract-example"}`, "--json")
	if contractRun.Status != "failed" || contractRun.Error.Code != "binding_not_found" {
		t.Fatalf("package.install default run = %#v, want L1 contract binding filtered by default L2 threshold", contractRun)
	}

	var executeAcquisition struct {
		State                string                    `json:"state"`
		TraceID              string                    `json:"trace_id"`
		CapabilitiesPromoted int                       `json:"capabilities_promoted"`
		BindingsPromoted     int                       `json:"bindings_promoted"`
		Providers            []e2etest.ProviderSummary `json:"providers"`
	}
	runDiscoveryForProviderPath(t, repo, env, calctlBin, providerPath, &executeAcquisition, "--capability-id", "system.validate", "--json")
	if executeAcquisition.State != "succeeded" || executeAcquisition.CapabilitiesPromoted != 1 || executeAcquisition.BindingsPromoted < 1 || executeAcquisition.TraceID == "" || len(executeAcquisition.Providers) != 1 {
		t.Fatalf("execute acquisition = %#v, want system.validate execute binding", executeAcquisition)
	}

	executeTrace := e2etest.ReadJSONFile[caltrace.Trace](t, filepath.Join(home, "discovery", executeAcquisition.TraceID, "trace.json"))
	assertLiveLLMCapabilityProbeAtLeast(t, executeTrace, "system.validate", core.VerifyLevelL2, core.VerifyMethodExecute)

	var inspectRun struct {
		Status   string             `json:"status"`
		Verified bool               `json:"verified"`
		Evidence []core.EvidenceRef `json:"evidence"`
		Outputs  map[string]any     `json:"outputs"`
	}
	reportPath := filepath.Join(temp, "doctor-report.json")
	e2etest.RunJSON(t, repo, env, &inspectRun, calctlBin, "runs", "create", "--capability-id", "system.validate", "--provider-id", executeAcquisition.Providers[0].ID, "--inputs-json", `{"target":`+strconv.Quote(reportPath)+`}`, "--verify", "--json")
	if inspectRun.Status != "succeeded" || !inspectRun.Verified || len(inspectRun.Evidence) == 0 {
		t.Fatalf("system.validate run = %#v, want verified L2+ nested command reuse", inspectRun)
	}
	report, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read doctor report: %v", err)
	}
	if !strings.Contains(string(report), `"status":"ok"`) {
		t.Fatalf("doctor report = %s, want status ok", report)
	}
}

func liveLLMEnv(t *testing.T, home string) []string {
	t.Helper()
	if os.Getenv("CAL_LIVE_LLM_E2E") != "1" {
		t.Skip("set CAL_LIVE_LLM_E2E=1 and CAL_LLM_* to run live LLM e2e")
	}
	required := []string{"CAL_LLM_API", "CAL_LLM_MODEL", "CAL_LLM_API_KEY"}
	for _, name := range required {
		if os.Getenv(name) == "" {
			t.Skipf("set %s to run live LLM e2e", name)
		}
	}
	if os.Getenv("CAL_LLM_API") != "chat_completions" {
		t.Skip("live Kimi/OpenAI-compatible e2e requires CAL_LLM_API=chat_completions")
	}
	return e2etest.WithHomeEnv(os.Environ(), home)
}

func assertLiveLLMTrace(t *testing.T, trace caltrace.Trace) string {
	t.Helper()
	if len(trace.Candidates) != 1 {
		t.Fatalf("trace candidates = %#v, want one candidate", trace.Candidates)
	}
	candidate := trace.Candidates[0]
	if candidate.CapabilityID == "" || candidate.Description == "" {
		t.Fatalf("candidate = %#v, want capability id and description", candidate)
	}
	args, ok := candidate.Execution.Spec[core.ExecutionSpecArgs].([]any)
	if !ok {
		t.Fatalf("candidate args = %#v, want JSON array", candidate.Execution.Spec[core.ExecutionSpecArgs])
	}
	wantArgs := []string{"make-pdf", "--in", "{{source}}", "--out", "{{target}}"}
	if len(args) != len(wantArgs) {
		t.Fatalf("candidate args = %#v, want %#v", args, wantArgs)
	}
	for index, want := range wantArgs {
		if args[index] != want {
			t.Fatalf("candidate args[%d] = %#v, want %q", index, args[index], want)
		}
	}
	if len(trace.Probes) != 1 || !trace.Probes[0].Passed || core.VerifyLevelRank(trace.Probes[0].Verify.Level) < core.VerifyLevelRank(core.VerifyLevelL1) {
		t.Fatalf("trace probes = %#v, want passing L1+ verify probe", trace.Probes)
	}
	promotions := e2etest.TracePromotions(trace)
	if len(promotions) != 1 || promotions[0].CapabilityID != candidate.CapabilityID || promotions[0].BindingID == "" {
		t.Fatalf("trace promotions = %#v, want %s promotion", promotions, candidate.CapabilityID)
	}
	return candidate.CapabilityID
}

func assertLiveLLMMultiCapTrace(t *testing.T, trace caltrace.Trace) (string, string) {
	t.Helper()
	if len(trace.Candidates) != 2 || len(trace.Probes) != 2 || len(e2etest.TracePromotions(trace)) != 2 {
		t.Fatalf("trace = %#v, want two candidates, probes, and promotions", trace)
	}
	var pdfCapabilityID string
	var noteCapabilityID string
	for _, candidate := range trace.Candidates {
		if candidate.CapabilityID == "" || candidate.Description == "" {
			t.Fatalf("candidate = %#v, want capability id and description", candidate)
		}
		args, ok := candidate.Execution.Spec[core.ExecutionSpecArgs].([]any)
		if !ok || len(args) == 0 {
			t.Fatalf("candidate args = %#v, want non-empty JSON array", candidate.Execution.Spec[core.ExecutionSpecArgs])
		}
		switch args[0] {
		case "make-pdf":
			pdfCapabilityID = candidate.CapabilityID
		case "write-note":
			noteCapabilityID = candidate.CapabilityID
		default:
			t.Fatalf("candidate = %#v, want make-pdf or write-note execution", candidate)
		}
	}
	if pdfCapabilityID == "" || noteCapabilityID == "" || pdfCapabilityID == noteCapabilityID {
		t.Fatalf("trace candidates = %#v, want distinct PDF and note capabilities", trace.Candidates)
	}
	for _, probe := range trace.Probes {
		if probe.CandidateIndex < 0 || probe.CandidateIndex >= len(trace.Candidates) {
			t.Fatalf("probe = %#v, want valid candidate index", probe)
		}
		capabilityID := trace.Candidates[probe.CandidateIndex].CapabilityID
		if !probe.Passed || core.VerifyLevelRank(probe.Verify.Level) < core.VerifyLevelRank(core.VerifyLevelL1) {
			t.Fatalf("probe = %#v, want passing L1+ verify for %s", probe, capabilityID)
		}
	}
	for _, promotion := range e2etest.TracePromotions(trace) {
		if promotion.CandidateIndex < 0 || promotion.CandidateIndex >= len(trace.Candidates) {
			t.Fatalf("promotion = %#v, want valid candidate index", promotion)
		}
		capabilityID := trace.Candidates[promotion.CandidateIndex].CapabilityID
		if promotion.CapabilityID != capabilityID || promotion.CapabilityAction != "created" || promotion.BindingAction != "created" || promotion.BindingID == "" {
			t.Fatalf("promotion = %#v, want created promotion for %s", promotion, capabilityID)
		}
	}
	return pdfCapabilityID, noteCapabilityID
}

func assertTraceVerifyForCapability(t *testing.T, trace caltrace.Trace, capabilityID string) {
	t.Helper()
	for _, probe := range trace.Probes {
		if probe.CandidateIndex < 0 || probe.CandidateIndex >= len(trace.Candidates) {
			continue
		}
		if trace.Candidates[probe.CandidateIndex].CapabilityID == capabilityID && probe.Passed && core.VerifyLevelRank(probe.Verify.Level) >= core.VerifyLevelRank(core.VerifyLevelL1) {
			return
		}
	}
	t.Fatalf("trace probes = %#v, missing passing verify for %s", trace.Probes, capabilityID)
}

func assertLiveLLMCapabilityProbe(t *testing.T, trace caltrace.Trace, capabilityID string, level core.VerifyLevel, method core.VerifyMethod) {
	t.Helper()
	for _, probe := range trace.Probes {
		if probe.CandidateIndex < 0 || probe.CandidateIndex >= len(trace.Candidates) {
			continue
		}
		if trace.Candidates[probe.CandidateIndex].CapabilityID != capabilityID {
			continue
		}
		if !probe.Passed || probe.Verify.Level != level || probe.Verify.Method != method {
			t.Fatalf("probe for %s = %#v, want passed %s %s", capabilityID, probe, level, method)
		}
		if len(probe.Evidence) != 1 {
			t.Fatalf("probe for %s evidence = %#v, want one evidence", capabilityID, probe.Evidence)
		}
		return
	}
	t.Fatalf("trace probes = %#v, missing probe for %s", trace.Probes, capabilityID)
}

func assertLiveLLMCapabilityProbeAtLeast(t *testing.T, trace caltrace.Trace, capabilityID string, level core.VerifyLevel, method core.VerifyMethod) {
	t.Helper()
	for _, probe := range trace.Probes {
		if probe.CandidateIndex < 0 || probe.CandidateIndex >= len(trace.Candidates) {
			continue
		}
		if trace.Candidates[probe.CandidateIndex].CapabilityID != capabilityID {
			continue
		}
		if !probe.Passed || core.VerifyLevelRank(probe.Verify.Level) < core.VerifyLevelRank(level) || probe.Verify.Method != method {
			t.Fatalf("probe for %s = %#v, want passed %s+ %s", capabilityID, probe, level, method)
		}
		if len(probe.Evidence) == 0 {
			t.Fatalf("probe for %s evidence = %#v, want evidence", capabilityID, probe.Evidence)
		}
		return
	}
	t.Fatalf("trace probes = %#v, missing probe for %s", trace.Probes, capabilityID)
}

func writeLiveLLMSeedConvertProposal(t *testing.T, path, capabilityID string) string {
	t.Helper()
	content := `{
  "metadata": {"source": "replay", "prompt_version": "test-v1", "model": "fixture", "schema_version": "proposal.v1"},
  "candidates": [{
    "capability_id": "` + capabilityID + `",
    "description": "Convert text into another document representation.",
    "execution": {
      "kind": "cli",
      "spec": {"args": ["make-pdf", "--in", "{{source}}", "--out", "{{target}}"]}
    }
  }],
  "probe_plans": [{
    "candidate_index": 0,
    "inputs": {"target": "{{workdir}}/output.pdf"},
    "fixtures": [{"input": "source", "filename": "input.txt", "content": "hello\n"}],
    "verify": {"level":"L2","method":"execute","checks":[{"subject":{"type":"file","input":"target"},"predicate":"format","params":{"format":"pdf"}}]}
  }]
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write live llm seed proposal: %v", err)
	}
	return path
}

func writeLiveLLMExporter(t *testing.T, path string) {
	t.Helper()
	script := `#!/bin/sh
if [ "$1" = "--help" ] || [ "$1" = "help" ]; then
  echo "Live LLM Exporter"
  echo "Usage: live-llm-exporter make-pdf --in <input-text> --out <output.pdf>"
  echo "Converts a UTF-8 text file into a valid PDF document."
  exit 0
fi
if [ "$1" = "make-pdf" ]; then
  source=""
  target=""
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --in)
        source="$2"
        shift 2
        ;;
      --out)
        target="$2"
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done
  if [ -z "$source" ] || [ -z "$target" ]; then
    exit 2
  fi
  ` + e2etest.WriteParseablePDFCommand() + `
  exit $?
fi
exit 64
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write live llm exporter: %v", err)
	}
}

func writeLiveLLMMultiCapabilityExporter(t *testing.T, path string) {
	t.Helper()
	script := `#!/bin/sh
if [ "$1" = "--help" ] || [ "$1" = "help" ]; then
  echo "Live Multi Tool"
  echo "Usage: live-multi-tool make-pdf --in <input.txt> --out <output.pdf>"
  echo "Usage: live-multi-tool write-note --in <input.txt> --out <output.txt>"
  echo "Commands:"
  echo "  make-pdf      Converts a UTF-8 text file into a valid PDF document."
  echo "  write-note    Writes a UTF-8 text note file."
  exit 0
fi
if [ "$1" = "make-pdf" ]; then
  source=""
  target=""
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --in)
        source="$2"
        shift 2
        ;;
      --out)
        target="$2"
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done
  if [ -z "$target" ]; then
    exit 2
  fi
  ` + e2etest.WriteParseablePDFCommand() + `
  exit $?
fi
if [ "$1" = "write-note" ]; then
  source=""
  target=""
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --in)
        source="$2"
        shift 2
        ;;
      --out)
        target="$2"
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done
  if [ -z "$target" ]; then
    exit 2
  fi
  if [ -n "$source" ] && [ -f "$source" ]; then
    cat "$source" > "$target"
  else
    printf 'hello\n' > "$target"
  fi
  exit $?
fi
exit 64
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write live llm multi-capability exporter: %v", err)
	}
}

func writeLiveLLMMarkerWriter(t *testing.T, path string) {
	t.Helper()
	script := `#!/bin/sh
if [ "$1" = "--help" ] || [ "$1" = "help" ]; then
  echo "Live Marker Tool"
  echo "Usage: live-marker-tool write-marker --in <input.txt> --out <output.txt>"
  echo "Commands:"
  echo "  write-marker    Writes the literal text CAL_PROBE_OK to the output file."
  echo "A correct output must contain CAL_PROBE_OK."
  exit 0
fi
if [ "$1" = "write-marker" ]; then
  target=""
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --out)
        target="$2"
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done
  if [ -z "$target" ]; then
    exit 2
  fi
  printf 'CAL_PROBE_OK\n' > "$target"
  exit $?
fi
exit 64
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write live llm marker writer: %v", err)
	}
}

func writeLiveLLMNestedPackageManager(t *testing.T, path string) {
	t.Helper()
	script := `#!/bin/sh
if [ "$1" = "--help" ] || [ "$1" = "help" ]; then
  echo "Live Nested Package Manager"
  echo "Usage:"
  echo "  live-nested-pm package install <name>"
  echo "  live-nested-pm package remove <name>"
  echo "  live-nested-pm package update"
  echo "  live-nested-pm package upgrade"
  echo "  live-nested-pm system doctor --json --output <report.json>"
  echo "Commands:"
  echo "  package install <name>   Install a package. Modifies local package state and may contact remote registries."
  echo "  package remove <name>    Remove an installed package. Modifies local package state."
  echo "  package update           Refresh package metadata from remote registries and modifies local metadata state."
  echo "  package upgrade          Upgrade installed packages and modifies local package state."
  echo "  system doctor --json --output <report.json>  Validate local package-manager configuration without changing state. Writes a JSON report with status=\"ok\" and checks[]."
  exit 0
fi
if [ "$1" = "system" ] && [ "$2" = "doctor" ]; then
  target=""
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --output)
        target="$2"
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done
  if [ -z "$target" ]; then
    exit 2
  fi
  printf '%s\n' '{"status":"ok","checks":[{"name":"prefix","result":"ok"},{"name":"cache","result":"ok"}]}' > "$target"
  printf '%s\n' "wrote $target"
  exit $?
fi
if [ "$1" = "package" ]; then
  case "$2" in
    install|remove|update|upgrade)
      printf '%s\n' "refusing to run state-changing package command in test fixture" >&2
      exit 70
      ;;
  esac
fi
exit 64
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write live llm nested package manager: %v", err)
	}
}
