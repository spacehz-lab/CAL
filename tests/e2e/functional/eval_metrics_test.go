package e2e

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
	e2etest "github.com/spacehz-lab/cal/internal/testsupport/e2e"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestExperimentEvalClosedLoopReportsAcquisitionAndReuse(t *testing.T) {
	repo, calctlBin, caldBin := functionalBinaries(t)
	temp := t.TempDir()

	home := filepath.Join(temp, "home")
	providerPath := filepath.Join(temp, "fake-exporter")
	e2etest.WriteFakeExporter(t, providerPath, e2etest.WriteParseablePDFCommand())

	env := e2etest.WithHomeEnv(os.Environ(), home)
	e2etest.StartCald(t, repo, env, caldBin)

	var firstScan struct {
		State                string                    `json:"state"`
		TraceID              string                    `json:"trace_id"`
		CapabilitiesPromoted int                       `json:"capabilities_promoted"`
		BindingsPromoted     int                       `json:"bindings_promoted"`
		Providers            []e2etest.ProviderSummary `json:"providers"`
	}
	runDiscoveryForProviderPath(t, repo, env, calctlBin, providerPath, &firstScan, "--mode", "rules", "--json")
	if firstScan.State != "succeeded" || firstScan.CapabilitiesPromoted != 1 || firstScan.BindingsPromoted != 1 || len(firstScan.Providers) != 1 {
		t.Fatalf("first scan = %#v, want created capability and binding", firstScan)
	}
	e2etest.AssertPromotionAction(t, home, firstScan.TraceID, "created", "created")

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
	e2etest.RunJSON(t, repo, env, &runSuccess, calctlBin, "runs", "create", "--capability-id", "document.convert", "--inputs-json", `{"source":`+strconv.Quote(source)+`,"target":`+strconv.Quote(target)+`}`, "--verify", "--json")
	if runSuccess.Status != "succeeded" || !runSuccess.Verified || len(runSuccess.Evidence) != 1 {
		t.Fatalf("run success = %#v, want verified reuse with evidence", runSuccess)
	}

	plainTarget := filepath.Join(temp, "plain-target.pdf")
	var plainRun struct {
		Status   string             `json:"status"`
		Verified bool               `json:"verified"`
		Evidence []core.EvidenceRef `json:"evidence"`
	}
	e2etest.RunJSON(t, repo, env, &plainRun, calctlBin, "runs", "create", "--capability-id", "document.convert", "--inputs-json", `{"source":`+strconv.Quote(source)+`,"target":`+strconv.Quote(plainTarget)+`}`, "--json")
	if plainRun.Status != "succeeded" || plainRun.Verified || len(plainRun.Evidence) != 0 {
		t.Fatalf("plain run = %#v, want unverified success without evidence", plainRun)
	}
	if _, err := os.Stat(plainTarget); err != nil {
		t.Fatalf("plain run target missing: %v", err)
	}

	providerID := firstScan.Providers[0].ID
	var secondScan struct {
		State                string `json:"state"`
		TraceID              string `json:"trace_id"`
		CapabilitiesPromoted int    `json:"capabilities_promoted"`
		BindingsPromoted     int    `json:"bindings_promoted"`
	}
	e2etest.RunJSON(t, repo, env, &secondScan, calctlBin, "discovery", "run", "--provider-id", providerID, "--mode", "rules", "--json")
	if secondScan.State != "succeeded" || secondScan.CapabilitiesPromoted != 0 || secondScan.BindingsPromoted != 1 {
		t.Fatalf("second scan = %#v, want reused capability and refreshed binding", secondScan)
	}
	e2etest.AssertPromotionAction(t, home, secondScan.TraceID, "reused", "updated")

	var metrics e2etest.EvalMetricsOutput
	e2etest.RunJSON(t, repo, env, &metrics, calctlBin, "eval", "--json")
	if metrics.Summary.Providers != 1 || metrics.Summary.Capabilities != 1 || metrics.Summary.Bindings != 1 || metrics.Summary.PromotedBindings != 1 || metrics.Summary.Traces != 2 || metrics.Summary.Runs != 2 {
		t.Fatalf("eval summary = %#v, want closed-loop record counts", metrics.Summary)
	}
	if metrics.Acquisition.AttemptCount != 2 || metrics.Acquisition.CompletedCount != 2 || metrics.Acquisition.PromotionCount != 2 || metrics.Acquisition.CandidateCount != 2 || metrics.Acquisition.ProbePassCount != 2 {
		t.Fatalf("eval acquisition counts = %#v, want two successful acquisition attempts", metrics.Acquisition)
	}
	if metrics.Acquisition.CapabilityCreatedCount != 1 || metrics.Acquisition.CapabilityReusedCount != 1 || metrics.Acquisition.BindingCreatedCount != 1 || metrics.Acquisition.BindingUpdatedCount != 1 {
		t.Fatalf("eval promotion actions = %#v, want created/reused and created/updated evidence", metrics.Acquisition)
	}
	if metrics.Acquisition.BindingPromotionRate != 1 || metrics.Acquisition.ProbeSuccessRate != 1 {
		t.Fatalf("eval acquisition rates = %#v, want perfect closed-loop acquisition rates", metrics.Acquisition)
	}
	if metrics.Reuse.RunCount != 2 || metrics.Reuse.RunSuccessCount != 2 || metrics.Reuse.RunFailureCount != 0 || metrics.Reuse.VerifiedRunCount != 1 {
		t.Fatalf("eval reuse counts = %#v, want one verified run and one plain successful run", metrics.Reuse)
	}
	if metrics.Reuse.RunSuccessRate != 1 || metrics.Reuse.VerifiedSuccessRate != 1 || metrics.Reuse.VerifierFailureRate != 0 {
		t.Fatalf("eval reuse = %#v, want verified successful reuse", metrics.Reuse)
	}
}

func TestRuntimeRunFailureIsPersistedInEval(t *testing.T) {
	repo, calctlBin, caldBin := functionalBinaries(t)
	temp := t.TempDir()

	home := filepath.Join(temp, "home")
	providerPath := filepath.Join(temp, "fake-exporter")
	e2etest.WriteFakeExporter(t, providerPath, `if [ ! -f "$source" ]; then
  printf '%s\n' 'source file missing' >&2
  exit 73
fi
`+e2etest.WriteParseablePDFCommand())
	env := e2etest.WithHomeEnv(os.Environ(), home)
	e2etest.StartCald(t, repo, env, caldBin)

	var acquisition struct {
		State                string `json:"state"`
		TraceID              string `json:"trace_id"`
		CapabilitiesPromoted int    `json:"capabilities_promoted"`
		BindingsPromoted     int    `json:"bindings_promoted"`
	}
	runDiscoveryForProviderPath(t, repo, env, calctlBin, providerPath, &acquisition, "--mode", "rules", "--json")
	if acquisition.State != "succeeded" || acquisition.CapabilitiesPromoted != 1 || acquisition.BindingsPromoted != 1 || acquisition.TraceID == "" {
		t.Fatalf("acquisition = %#v, want promoted binding before runtime failure", acquisition)
	}

	source := filepath.Join(temp, "missing-source.txt")
	target := filepath.Join(temp, "target.pdf")
	var runFailure struct {
		Status string `json:"status"`
		Error  struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	e2etest.RunFailJSON(t, repo, env, &runFailure, calctlBin, "runs", "create", "--capability-id", "document.convert", "--inputs-json", `{"source":`+strconv.Quote(source)+`,"target":`+strconv.Quote(target)+`}`, "--json")
	if runFailure.Status != "failed" || runFailure.Error.Code != "execution_failed" {
		t.Fatalf("run failure = %#v, want execution_failed", runFailure)
	}

	var metrics e2etest.EvalMetricsOutput
	e2etest.RunJSON(t, repo, env, &metrics, calctlBin, "eval", "--json")
	if metrics.Summary.Runs != 1 {
		t.Fatalf("eval summary = %#v, want one persisted failed run", metrics.Summary)
	}
	if metrics.Reuse.RunCount != 1 || metrics.Reuse.RunSuccessCount != 0 || metrics.Reuse.RunFailureCount != 1 || metrics.Reuse.VerifiedRunCount != 0 {
		t.Fatalf("eval reuse = %#v, want one unverified execution failure", metrics.Reuse)
	}
}

func TestRuntimeVerifyFailureIsPersistedInEval(t *testing.T) {
	repo, calctlBin, caldBin := functionalBinaries(t)
	temp := t.TempDir()

	home := filepath.Join(temp, "home")
	providerPath := filepath.Join(temp, "conditional-pdf")
	e2etest.WriteConditionalPDFExporter(t, providerPath)
	proposalPath := e2etest.WriteReplayProposal(t, filepath.Join(temp, "proposal.json"))
	env := e2etest.WithHomeEnv(os.Environ(), home)
	e2etest.StartCald(t, repo, env, caldBin)

	var acquisition struct {
		State                string `json:"state"`
		TraceID              string `json:"trace_id"`
		CapabilitiesPromoted int    `json:"capabilities_promoted"`
		BindingsPromoted     int    `json:"bindings_promoted"`
	}
	runDiscoveryForProviderPath(t, repo, env, calctlBin, providerPath, &acquisition, "--proposal-path", proposalPath, "--json")
	if acquisition.State != "succeeded" || acquisition.CapabilitiesPromoted != 1 || acquisition.BindingsPromoted != 1 || acquisition.TraceID == "" {
		t.Fatalf("acquisition = %#v, want promoted binding before runtime verifier failure", acquisition)
	}

	source := filepath.Join(temp, "bad-source.txt")
	target := filepath.Join(temp, "bad-target.pdf")
	if err := os.WriteFile(source, []byte("bad-runtime-pdf\n"), 0o644); err != nil {
		t.Fatalf("write bad source: %v", err)
	}
	var runFailure struct {
		ID       string `json:"id"`
		Status   string `json:"status"`
		Verified bool   `json:"verified"`
		Error    struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	e2etest.RunFailJSON(t, repo, env, &runFailure, calctlBin, "runs", "create", "--capability-id", "document.convert", "--inputs-json", `{"source":`+strconv.Quote(source)+`,"target":`+strconv.Quote(target)+`}`, "--verify", "--json")
	if runFailure.Status != "failed" || runFailure.Verified || runFailure.Error.Code != "verification_failed" || runFailure.ID == "" {
		t.Fatalf("run failure = %#v, want persisted verification_failed run", runFailure)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("target missing after verifier failure: %v", err)
	}

	var storedRun struct {
		ID       string `json:"id"`
		Status   string `json:"status"`
		Verified bool   `json:"verified"`
		Error    struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	e2etest.RunJSON(t, repo, env, &storedRun, calctlBin, "runs", "get", "--run-id", runFailure.ID, "--json")
	if storedRun.ID != runFailure.ID || storedRun.Status != "failed" || storedRun.Verified || storedRun.Error.Code != "verification_failed" {
		t.Fatalf("stored run = %#v, want persisted verification_failed run %s", storedRun, runFailure.ID)
	}

	var metrics e2etest.EvalMetricsOutput
	e2etest.RunJSON(t, repo, env, &metrics, calctlBin, "eval", "--json")
	if metrics.Summary.Runs != 1 {
		t.Fatalf("eval summary = %#v, want one persisted verifier-failed run", metrics.Summary)
	}
	if metrics.Reuse.RunCount != 1 || metrics.Reuse.RunSuccessCount != 0 || metrics.Reuse.RunFailureCount != 1 || metrics.Reuse.VerifiedRunCount != 1 || metrics.Reuse.VerifierFailCount != 1 {
		t.Fatalf("eval reuse = %#v, want one verifier-failed run", metrics.Reuse)
	}
	if metrics.Reuse.VerifierFailureRate != 1 {
		t.Fatalf("eval verifier failure rate = %v, want 1", metrics.Reuse.VerifierFailureRate)
	}
}

func TestReplayProposalAcquisitionPromotesMultipleCapabilities(t *testing.T) {
	repo, calctlBin, caldBin := functionalBinaries(t)
	temp := t.TempDir()

	home := filepath.Join(temp, "home")
	providerPath := filepath.Join(temp, "multi-exporter")
	e2etest.WriteMultiCapabilityExporter(t, providerPath)
	proposalPath := e2etest.WriteMultiCapabilityProposal(t, filepath.Join(temp, "proposal.json"))

	env := e2etest.WithHomeEnv(os.Environ(), home)
	e2etest.StartCald(t, repo, env, caldBin)

	var acquisition struct {
		State                string `json:"state"`
		TraceID              string `json:"trace_id"`
		CapabilitiesPromoted int    `json:"capabilities_promoted"`
		BindingsPromoted     int    `json:"bindings_promoted"`
	}
	runDiscoveryForProviderPath(t, repo, env, calctlBin, providerPath, &acquisition, "--proposal-path", proposalPath, "--json")
	if acquisition.State != "succeeded" || acquisition.CapabilitiesPromoted != 2 || acquisition.BindingsPromoted != 2 || acquisition.TraceID == "" {
		t.Fatalf("acquisition = %#v, want two promoted capabilities", acquisition)
	}

	trace := e2etest.ReadJSONFile[caltrace.Trace](t, filepath.Join(home, "discovery", acquisition.TraceID, "trace.json"))
	if len(trace.Candidates) != 2 || len(trace.Probes) != 2 || len(e2etest.TracePromotions(trace)) != 2 {
		t.Fatalf("trace = %#v, want two candidates, probes, and promotions", trace)
	}
	for index, promotion := range e2etest.TracePromotions(trace) {
		if promotion.CandidateIndex != index || promotion.CapabilityAction != "created" || promotion.BindingAction != "created" {
			t.Fatalf("promotion[%d] = %#v, want created promotion for matching candidate", index, promotion)
		}
	}
	noteProbePassed := false
	for _, probe := range trace.Probes {
		if probe.CandidateIndex == 1 {
			noteProbePassed = probe.Passed && core.VerifyLevelRank(probe.Verify.Level) >= core.VerifyLevelRank(core.VerifyLevelL2)
		}
	}
	if !noteProbePassed {
		t.Fatalf("trace probes = %#v, want passing L2+ verify for text.write candidate", trace.Probes)
	}

	noteTarget := filepath.Join(temp, "note.txt")
	var runSuccess struct {
		Status   string             `json:"status"`
		Verified bool               `json:"verified"`
		Evidence []core.EvidenceRef `json:"evidence"`
	}
	e2etest.RunJSON(t, repo, env, &runSuccess, calctlBin, "runs", "create", "--capability-id", "text.write", "--inputs-json", `{"target":`+strconv.Quote(noteTarget)+`}`, "--verify", "--json")
	if runSuccess.Status != "succeeded" || !runSuccess.Verified || len(runSuccess.Evidence) == 0 {
		t.Fatalf("text.write run = %#v, want verified reuse", runSuccess)
	}
	content, err := os.ReadFile(noteTarget)
	if err != nil {
		t.Fatalf("read note target: %v", err)
	}
	if string(content) != "hello\n" {
		t.Fatalf("note target content = %q, want hello", content)
	}

	var metrics e2etest.EvalMetricsOutput
	e2etest.RunJSON(t, repo, env, &metrics, calctlBin, "eval", "--json")
	if metrics.Summary.Capabilities != 2 || metrics.Summary.Bindings != 2 || metrics.Summary.Runs != 1 || metrics.Acquisition.PromotionCount != 2 || metrics.Acquisition.CandidateCount != 2 || metrics.Acquisition.ProbePassCount != 2 {
		t.Fatalf("eval = %#v, want multi-cap acquisition counts", metrics)
	}
	if metrics.Acquisition.BindingPromotionRate != 1 || metrics.Acquisition.ProbeSuccessRate != 1 {
		t.Fatalf("eval acquisition rates = %#v, want successful multi-cap acquisition rates", metrics.Acquisition)
	}
	if metrics.Reuse.RunCount != 1 || metrics.Reuse.RunSuccessCount != 1 || metrics.Reuse.VerifiedRunCount != 1 || metrics.Reuse.VerifiedSuccessRate != 1 {
		t.Fatalf("eval reuse = %#v, want successful second capability reuse", metrics.Reuse)
	}

	useTarget := filepath.Join(temp, "use-note.txt")
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
	e2etest.RunJSON(t, repo, env, &useSuccess, calctlBin, "use", "--intent", "write a text file artifact", "--inputs-json", `{"target":`+strconv.Quote(useTarget)+`}`, "--verify", "--json")
	if useSuccess.Status != "succeeded" || useSuccess.Selection.CapabilityID != "text.write" || useSuccess.Selection.BindingID == "" {
		t.Fatalf("use success = %#v, want text.write selection", useSuccess)
	}
	if useSuccess.Run.Status != "succeeded" || !useSuccess.Run.Verified || useSuccess.Run.CapabilityID != "text.write" || useSuccess.Run.BindingID != useSuccess.Selection.BindingID {
		t.Fatalf("use run = %#v, want verified selected text.write binding", useSuccess.Run)
	}
	content, err = os.ReadFile(useTarget)
	if err != nil {
		t.Fatalf("read use target: %v", err)
	}
	if string(content) != "hello\n" {
		t.Fatalf("use target content = %q, want hello", content)
	}

	e2etest.RunJSON(t, repo, env, &metrics, calctlBin, "eval", "--json")
	if metrics.Summary.Runs != 2 || metrics.Reuse.RunCount != 2 || metrics.Reuse.RunSuccessCount != 2 || metrics.Reuse.VerifiedRunCount != 2 || metrics.Reuse.VerifiedSuccessRate != 1 {
		t.Fatalf("eval after use = %#v, want direct run and intent use reuse counts", metrics)
	}
}
