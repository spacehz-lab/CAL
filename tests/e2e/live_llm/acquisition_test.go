package e2e

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/contract"
	"github.com/spacehz-lab/cal/internal/model"
	runpkg "github.com/spacehz-lab/cal/internal/run"
)

func TestLiveLLMAcquisitionPromotesNewCapability(t *testing.T) {
	repo, calctlBin, caldBin := liveBinaries(t)
	workspace := liveLLMWorkspace(t)
	env := liveLLMEnv(t, workspace.home)
	startCald(t, repo, env, caldBin)

	providerPath := filepath.Join(workspace.temp, "live-exporter")
	writeLiveLLMExporter(t, providerPath)

	_, acquisition := runAcquisitionForProviderPath(t, repo, env, calctlBin, providerPath, "--json")
	trace := assertAcquisitionCompleted(t, &acquisition, 1, 1)
	storedTrace := assertTraceStored(t, workspace.home, acquisition.TraceID)
	capabilityID := assertLiveLLMTrace(t, trace)
	assertLiveLLMTrace(t, storedTrace)
	capability := readJSONFile[model.Capability](t, filepath.Join(workspace.home, "capabilities", capabilityID+".json"))
	if len(capability.Bindings) != 1 {
		t.Fatalf("capability = %#v, want one promoted binding", capability)
	}

	source := filepath.Join(workspace.temp, "source.txt")
	target := filepath.Join(workspace.temp, "target.pdf")
	if err := os.WriteFile(source, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	var runResponse contract.RunResponse
	runJSON(t, repo, env, &runResponse, calctlBin, "runs", "create", "--capability-id", capabilityID, "--inputs-json", `{"source":`+strconv.Quote(source)+`,"target":`+strconv.Quote(target)+`}`, "--verify", "--json")
	run := assertRunSucceeded(t, runResponse)
	if !run.Verified || len(run.Evidence) == 0 {
		t.Fatalf("run = %#v, want verified live LLM capability reuse", run)
	}

	useTarget := filepath.Join(workspace.temp, "use-target.pdf")
	var useResponse contract.UseResponse
	runJSON(t, repo, env, &useResponse, calctlBin, "use", "convert a utf-8 text file into a valid pdf document", "--inputs-json", `{"source":`+strconv.Quote(source)+`,"target":`+strconv.Quote(useTarget)+`}`, "--verify", "--min-verify-level", "L1", "--json")
	if useResponse.Status != model.RunStatusSucceeded || useResponse.Selection == nil || useResponse.Selection.CapabilityID != capabilityID || useResponse.Selection.BindingID == "" {
		t.Fatalf("use response = %#v, want selected live LLM capability %s", useResponse, capabilityID)
	}
	if useResponse.Run == nil || useResponse.Run.Status != model.RunStatusSucceeded || !useResponse.Run.Verified || useResponse.Run.BindingID != useResponse.Selection.BindingID || useResponse.Run.CapabilityID != capabilityID {
		t.Fatalf("use run = %#v, want verified selected live LLM binding", useResponse.Run)
	}
	if _, err := os.Stat(useTarget); err != nil {
		t.Fatalf("use target missing: %v", err)
	}

	var metrics contract.EvalResponse
	runJSON(t, repo, env, &metrics, calctlBin, "eval", "--json")
	if metrics.Capability.Capabilities != 1 || metrics.Capability.Bindings != 1 || metrics.Capability.PromotedBindings != 1 || metrics.Acquisition.Traces.Total != 1 || metrics.Reuse.Runs.Total != 2 || metrics.Reuse.Verified != 2 {
		t.Fatalf("eval = %#v, want live LLM new capability closed-loop records", metrics)
	}
}

func TestLiveLLMAcquisitionReusesExistingCapability(t *testing.T) {
	repo, calctlBin, caldBin := liveBinaries(t)
	workspace := liveLLMWorkspace(t)
	env := liveLLMEnv(t, workspace.home)
	startCald(t, repo, env, caldBin)

	const seededCapabilityID = "text.convert"
	seedPath := filepath.Join(workspace.temp, "seed-exporter")
	writeLiveLLMExporter(t, seedPath)
	seedProposalPath := writeLiveLLMSeedConvertProposal(t, filepath.Join(workspace.temp, "seed-proposal.json"), seededCapabilityID)
	_, seed := runAcquisitionForProviderPath(t, repo, env, calctlBin, seedPath, "--mode", "replay", "--proposal-path", seedProposalPath, "--json")
	assertAcquisitionCompleted(t, &seed, 1, 1)

	providerPath := filepath.Join(workspace.temp, "live-exporter")
	writeLiveLLMExporter(t, providerPath)
	provider, acquisition := runAcquisitionForProviderPath(t, repo, env, calctlBin, providerPath, "--json")
	trace := assertAcquisitionCompleted(t, &acquisition, 1, 1)
	assertLiveLLMTrace(t, trace)
	if len(trace.Promotions) != 1 || trace.Promotions[0].CapabilityAction != "reused" || trace.Promotions[0].BindingAction != "created" {
		t.Fatalf("trace promotions = %#v, want reused capability and created binding", trace.Promotions)
	}
	capability := readJSONFile[model.Capability](t, filepath.Join(workspace.home, "capabilities", seededCapabilityID+".json"))
	if len(capability.Bindings) != 2 {
		t.Fatalf("capability = %#v, want seeded binding plus live LLM binding", capability)
	}

	source := filepath.Join(workspace.temp, "source.txt")
	target := filepath.Join(workspace.temp, "target.pdf")
	if err := os.WriteFile(source, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	var runResponse contract.RunResponse
	runJSON(t, repo, env, &runResponse, calctlBin, "runs", "create", "--capability-id", seededCapabilityID, "--provider-id", provider.ID, "--inputs-json", `{"source":`+strconv.Quote(source)+`,"target":`+strconv.Quote(target)+`}`, "--verify", "--json")
	run := assertRunSucceeded(t, runResponse)
	if !run.Verified || len(run.Evidence) == 0 {
		t.Fatalf("run = %#v, want verification evidence", run)
	}

	var metrics contract.EvalResponse
	runJSON(t, repo, env, &metrics, calctlBin, "eval", "--json")
	if metrics.Capability.Capabilities != 1 || metrics.Capability.Bindings != 2 || metrics.Capability.PromotedBindings != 2 || metrics.Acquisition.Traces.Total != 2 || metrics.Acquisition.Promotions.Total != 2 || metrics.Acquisition.Promotions.Bindings != 2 || metrics.Reuse.Runs.Total != 1 || metrics.Reuse.Verified != 1 {
		t.Fatalf("eval = %#v, want live LLM capability reuse records", metrics)
	}
}

func TestLiveLLMAcquisitionPromotesMultipleCapabilities(t *testing.T) {
	repo, calctlBin, caldBin := liveBinaries(t)
	workspace := liveLLMWorkspace(t)
	env := liveLLMEnv(t, workspace.home)
	startCald(t, repo, env, caldBin)

	providerPath := filepath.Join(workspace.temp, "live-multi-tool")
	writeLiveLLMMultiCapabilityExporter(t, providerPath)
	provider, acquisition := runAcquisitionForProviderPath(t, repo, env, calctlBin, providerPath, "--json")
	trace := assertAcquisitionCompleted(t, &acquisition, 2, 2)
	pdfCapabilityID, noteCapabilityID := assertLiveLLMMultiCapTrace(t, trace)
	assertTraceVerifyForCapability(t, trace, pdfCapabilityID)
	assertTraceVerifyForCapability(t, trace, noteCapabilityID)
	for _, capabilityID := range []string{pdfCapabilityID, noteCapabilityID} {
		capability := readJSONFile[model.Capability](t, filepath.Join(workspace.home, "capabilities", capabilityID+".json"))
		if len(capability.Bindings) != 1 {
			t.Fatalf("capability %s = %#v, want one promoted binding", capabilityID, capability)
		}
	}

	pdfSource := filepath.Join(workspace.temp, "source.txt")
	pdfTarget := filepath.Join(workspace.temp, "target.pdf")
	if err := os.WriteFile(pdfSource, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write pdf source: %v", err)
	}
	var pdfRun contract.RunResponse
	runJSON(t, repo, env, &pdfRun, calctlBin, "runs", "create", "--capability-id", pdfCapabilityID, "--provider-id", provider.ID, "--inputs-json", `{"source":`+strconv.Quote(pdfSource)+`,"target":`+strconv.Quote(pdfTarget)+`}`, "--verify", "--json")
	if run := assertRunSucceeded(t, pdfRun); !run.Verified || len(run.Evidence) == 0 {
		t.Fatalf("pdf run = %#v, want verified %s reuse", run, pdfCapabilityID)
	}

	noteSource := filepath.Join(workspace.temp, "note-source.txt")
	noteTarget := filepath.Join(workspace.temp, "note.txt")
	if err := os.WriteFile(noteSource, []byte("note\n"), 0o644); err != nil {
		t.Fatalf("write note source: %v", err)
	}
	var noteRun contract.RunResponse
	runJSON(t, repo, env, &noteRun, calctlBin, "runs", "create", "--capability-id", noteCapabilityID, "--provider-id", provider.ID, "--inputs-json", `{"source":`+strconv.Quote(noteSource)+`,"target":`+strconv.Quote(noteTarget)+`}`, "--verify", "--json")
	if run := assertRunSucceeded(t, noteRun); !run.Verified || len(run.Evidence) == 0 {
		t.Fatalf("note run = %#v, want verified %s reuse", run, noteCapabilityID)
	}

	var metrics contract.EvalResponse
	runJSON(t, repo, env, &metrics, calctlBin, "eval", "--json")
	if metrics.Capability.Capabilities != 2 || metrics.Capability.Bindings != 2 || metrics.Acquisition.Traces.Total != 1 || metrics.Acquisition.Candidates != 2 || metrics.Acquisition.Probes.Passed != 2 || metrics.Reuse.Runs.Total != 2 || metrics.Reuse.Verified != 2 {
		t.Fatalf("eval = %#v, want live LLM multi-cap closed-loop records", metrics)
	}
}

func TestLiveLLMAcquisitionGeneratesVerifySpec(t *testing.T) {
	repo, calctlBin, caldBin := liveBinaries(t)
	workspace := liveLLMWorkspace(t)
	env := liveLLMEnv(t, workspace.home)
	startCald(t, repo, env, caldBin)

	providerPath := filepath.Join(workspace.temp, "live-marker-tool")
	writeLiveLLMMarkerWriter(t, providerPath)
	provider, acquisition := runAcquisitionForProviderPath(t, repo, env, calctlBin, providerPath, "--json")
	trace := assertAcquisitionCompleted(t, &acquisition, 1, 1)
	if len(trace.Candidates) != 1 || len(trace.Probes) != 1 || !trace.Probes[0].Passed || model.VerifyLevelRank(trace.Probes[0].Verify.Level) < model.VerifyLevelRank(model.VerifyLevelL1) {
		t.Fatalf("trace = %#v, want one passed verify-spec probe", trace)
	}

	source := filepath.Join(workspace.temp, "source.txt")
	target := filepath.Join(workspace.temp, "marker.txt")
	if err := os.WriteFile(source, []byte("ignored\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	var runResponse contract.RunResponse
	runJSON(t, repo, env, &runResponse, calctlBin, "runs", "create", "--capability-id", trace.Candidates[0].CapabilityID, "--provider-id", provider.ID, "--inputs-json", `{"source":`+strconv.Quote(source)+`,"target":`+strconv.Quote(target)+`}`, "--verify", "--json")
	run := assertRunSucceeded(t, runResponse)
	if !run.Verified || len(run.Evidence) == 0 {
		t.Fatalf("run = %#v, want verify-spec reuse", run)
	}

	var metrics contract.EvalResponse
	runJSON(t, repo, env, &metrics, calctlBin, "eval", "--json")
	if metrics.Capability.Capabilities != 1 || metrics.Capability.Bindings != 1 || metrics.Acquisition.Traces.Total != 1 || metrics.Reuse.Runs.Total != 1 || metrics.Reuse.Verified != 1 {
		t.Fatalf("eval = %#v, want verify-spec closed-loop records", metrics)
	}
}

func TestLiveLLMAcquisitionHandlesNestedCommandContractAndExecute(t *testing.T) {
	repo, calctlBin, caldBin := liveBinaries(t)
	workspace := liveLLMWorkspace(t)
	env := liveLLMEnv(t, workspace.home)
	startCald(t, repo, env, caldBin)

	providerPath := filepath.Join(workspace.temp, "live-nested-pm")
	writeLiveLLMNestedPackageManager(t, providerPath)

	provider, contractAcquisition := runAcquisitionForProviderPath(t, repo, env, calctlBin, providerPath, "--hint", "install a package", "--json")
	contractTrace := assertAcquisitionCompleted(t, &contractAcquisition, 1, 1)
	packageInstallID := liveLLMCapabilityIDForExecution(t, contractTrace, model.VerifyMethodContract, "package", "install")
	assertLiveLLMCapabilityProbe(t, contractTrace, packageInstallID, model.VerifyLevelL1, model.VerifyMethodContract)

	var contractRun contract.RunResponse
	runJSON(t, repo, env, &contractRun, calctlBin, "runs", "create", "--capability-id", packageInstallID, "--provider-id", provider.ID, "--inputs-json", `{"package":"cal-contract-example"}`, "--min-verify-level", "L2", "--json")
	if contractRun.Run == nil || contractRun.Run.Status != model.RunStatusFailed || contractRun.Run.Error == nil || contractRun.Run.Error.Code != runpkg.ErrorBindingNotFound {
		t.Fatalf("%s run = %#v, want L1 contract binding filtered by L2 threshold", packageInstallID, contractRun)
	}

	_, executeAcquisition := runAcquisitionForProviderPath(t, repo, env, calctlBin, providerPath, "--hint", "validate package manager configuration and write a json report", "--json")
	executeTrace := assertAcquisitionCompleted(t, &executeAcquisition, 1, 1)
	systemValidateID := liveLLMCapabilityIDForExecution(t, executeTrace, model.VerifyMethodExecute, "system", "doctor")
	assertLiveLLMCapabilityProbeAtLeast(t, executeTrace, systemValidateID, model.VerifyLevelL1, model.VerifyMethodExecute)

	reportPath := filepath.Join(workspace.temp, "doctor-report.json")
	reportInput := liveLLMOutputPathInput(t, executeTrace, systemValidateID)
	var inspectRun contract.RunResponse
	runJSON(t, repo, env, &inspectRun, calctlBin, "runs", "create", "--capability-id", systemValidateID, "--provider-id", provider.ID, "--inputs-json", `{"`+reportInput+`":`+strconv.Quote(reportPath)+`}`, "--verify", "--json")
	run := assertRunSucceeded(t, inspectRun)
	if !run.Verified || len(run.Evidence) == 0 {
		t.Fatalf("%s run = %#v, want verified nested command reuse", systemValidateID, run)
	}
	report, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read doctor report for input %s: %v", reportInput, err)
	}
	if !strings.Contains(string(report), `"status":"ok"`) {
		t.Fatalf("doctor report = %s, want status ok", report)
	}
}
