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

func TestControlledAcquisitionPromotesParseVerifiedBinding(t *testing.T) {
	repo, calctlBin, caldBin := functionalBinaries(t)
	temp := t.TempDir()

	home := filepath.Join(temp, "home")
	providerPath := filepath.Join(temp, "fake-exporter")
	e2etest.WriteFakeExporter(t, providerPath, e2etest.WriteParseablePDFCommand())

	env := e2etest.WithHomeEnv(os.Environ(), home)
	e2etest.StartCald(t, repo, env, caldBin)

	var acquisition struct {
		State                string                    `json:"state"`
		TraceID              string                    `json:"trace_id"`
		CapabilitiesPromoted int                       `json:"capabilities_promoted"`
		BindingsPromoted     int                       `json:"bindings_promoted"`
		Providers            []e2etest.ProviderSummary `json:"providers"`
	}
	runDiscoveryForProviderPath(t, repo, env, calctlBin, providerPath, &acquisition, "--mode", "rules", "--json")
	if acquisition.State != "succeeded" || acquisition.CapabilitiesPromoted != 1 || acquisition.BindingsPromoted != 1 || acquisition.TraceID == "" || len(acquisition.Providers) != 1 {
		t.Fatalf("acquisition discovery = %#v, want one promoted parse-verified binding", acquisition)
	}
	provider := acquisition.Providers[0]
	trace := e2etest.ReadJSONFile[caltrace.Trace](t, filepath.Join(home, "discovery", acquisition.TraceID, "trace.json"))
	if len(trace.Probes) != 1 {
		t.Fatalf("trace probes = %#v, want one probe", trace.Probes)
	}
	if !trace.Probes[0].Passed || core.VerifyLevelRank(trace.Probes[0].Verify.Level) < core.VerifyLevelRank(core.VerifyLevelL2) {
		t.Fatalf("trace probe = %#v, want passing L2+ verify", trace.Probes[0])
	}
	if len(trace.Probes[0].Evidence) != 1 {
		t.Fatalf("trace probe evidence = %#v, want one evidence", trace.Probes[0].Evidence)
	}

	capability := e2etest.ReadJSONFile[core.Capability](t, filepath.Join(home, "capabilities", "document.export_pdf.json"))
	if len(capability.Bindings) != 1 {
		t.Fatalf("capability bindings = %#v, want one binding", capability.Bindings)
	}
	binding := capability.Bindings[0]
	if binding.State != core.BindingStatePromoted || binding.ProviderID != provider.ID {
		t.Fatalf("binding = %#v, want promoted binding for provider %s", binding, provider.ID)
	}
	if binding.Verify == nil || core.VerifyLevelRank(binding.Verify.Level) < core.VerifyLevelRank(core.VerifyLevelL2) {
		t.Fatalf("binding verify = %#v, want L2+", binding.Verify)
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
	e2etest.RunJSON(t, repo, env, &runSuccess, calctlBin, "runs", "create", "--capability-id", "document.export_pdf", "--inputs-json", `{"source":`+strconv.Quote(source)+`,"target":`+strconv.Quote(target)+`}`, "--verify", "--json")
	if runSuccess.Status != "succeeded" || !runSuccess.Verified {
		t.Fatalf("run success = %#v, want verified success", runSuccess)
	}
	if len(runSuccess.Evidence) != 1 {
		t.Fatalf("run evidence = %#v, want one evidence", runSuccess.Evidence)
	}
}

func TestMarkerFreeFakeCLIAcquisitionPromotesParseVerifiedBinding(t *testing.T) {
	repo, calctlBin, caldBin := functionalBinaries(t)
	temp := t.TempDir()

	home := filepath.Join(temp, "home")
	providerPath := filepath.Join(temp, "fake-exporter")
	e2etest.WriteMarkerFreeFakeExporter(t, providerPath, e2etest.WriteParseablePDFCommand())

	env := e2etest.WithHomeEnv(os.Environ(), home)
	e2etest.StartCald(t, repo, env, caldBin)

	var acquisition struct {
		State                string                    `json:"state"`
		TraceID              string                    `json:"trace_id"`
		CapabilitiesPromoted int                       `json:"capabilities_promoted"`
		BindingsPromoted     int                       `json:"bindings_promoted"`
		Providers            []e2etest.ProviderSummary `json:"providers"`
	}
	runDiscoveryForProviderPath(t, repo, env, calctlBin, providerPath, &acquisition, "--mode", "rules", "--json")
	if acquisition.State != "succeeded" || acquisition.CapabilitiesPromoted != 1 || acquisition.BindingsPromoted != 1 || acquisition.TraceID == "" || len(acquisition.Providers) != 1 {
		t.Fatalf("acquisition discovery = %#v, want one promoted marker-free binding", acquisition)
	}
	trace := e2etest.ReadJSONFile[caltrace.Trace](t, filepath.Join(home, "discovery", acquisition.TraceID, "trace.json"))
	if len(trace.Observations) != 1 {
		t.Fatalf("trace observations = %#v, want one help observation", trace.Observations)
	}
	text, ok := trace.Observations[0].Content["text"].(string)
	if !ok {
		t.Fatalf("trace observation text = %#v, want string", trace.Observations[0].Content["text"])
	}
	if strings.Contains(text, "CAL_CAPABILITY") || strings.Contains(text, "CAL_COMMAND") {
		t.Fatalf("trace observation contains CAL marker, want marker-free help:\n%s", text)
	}
	if len(trace.Candidates) != 1 || trace.Candidates[0].Source != "rules:cli_help_export_pdf" {
		t.Fatalf("trace candidates = %#v, want marker-free export-pdf candidate", trace.Candidates)
	}
	args, ok := trace.Candidates[0].Execution.Spec["args"].([]any)
	if !ok {
		t.Fatalf("candidate args = %#v, want JSON array", trace.Candidates[0].Execution.Spec["args"])
	}
	wantArgs := []string{"export-pdf", "--source", "{{source}}", "--target", "{{target}}"}
	if len(args) != len(wantArgs) {
		t.Fatalf("candidate args len = %d, want %d", len(args), len(wantArgs))
	}
	for index, want := range wantArgs {
		if args[index] != want {
			t.Fatalf("candidate args[%d] = %#v, want %q", index, args[index], want)
		}
	}
	if len(trace.Probes) != 1 || !trace.Probes[0].Passed || core.VerifyLevelRank(trace.Probes[0].Verify.Level) < core.VerifyLevelRank(core.VerifyLevelL2) {
		t.Fatalf("trace probes = %#v, want passing L2+ probe", trace.Probes)
	}

	capability := e2etest.ReadJSONFile[core.Capability](t, filepath.Join(home, "capabilities", "document.export_pdf.json"))
	if len(capability.Bindings) != 1 || capability.Bindings[0].Verify == nil || core.VerifyLevelRank(capability.Bindings[0].Verify.Level) < core.VerifyLevelRank(core.VerifyLevelL2) {
		t.Fatalf("capability = %#v, want one L2+ verified binding", capability)
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
	e2etest.RunJSON(t, repo, env, &runSuccess, calctlBin, "runs", "create", "--capability-id", "document.export_pdf", "--inputs-json", `{"source":`+strconv.Quote(source)+`,"target":`+strconv.Quote(target)+`}`, "--verify", "--json")
	if runSuccess.Status != "succeeded" || !runSuccess.Verified {
		t.Fatalf("run success = %#v, want verified success", runSuccess)
	}
	if len(runSuccess.Evidence) != 1 {
		t.Fatalf("run evidence = %#v, want one evidence", runSuccess.Evidence)
	}
}

func TestReplayProposalAcquisitionPromotesParseVerifiedBinding(t *testing.T) {
	repo, calctlBin, caldBin := functionalBinaries(t)
	temp := t.TempDir()

	home := filepath.Join(temp, "home")
	providerPath := filepath.Join(temp, "proposal-exporter")
	e2etest.WriteProposalBackedExporter(t, providerPath)
	proposalPath := e2etest.WriteReplayProposal(t, filepath.Join(temp, "proposal.json"))

	env := e2etest.WithHomeEnv(os.Environ(), home)
	e2etest.StartCald(t, repo, env, caldBin)

	var acquisition struct {
		State                string                    `json:"state"`
		TraceID              string                    `json:"trace_id"`
		CapabilitiesPromoted int                       `json:"capabilities_promoted"`
		BindingsPromoted     int                       `json:"bindings_promoted"`
		Providers            []e2etest.ProviderSummary `json:"providers"`
	}
	runDiscoveryForProviderPath(t, repo, env, calctlBin, providerPath, &acquisition, "--proposal-path", proposalPath, "--json")
	if acquisition.State != "succeeded" || acquisition.CapabilitiesPromoted != 1 || acquisition.BindingsPromoted != 1 || acquisition.TraceID == "" || len(acquisition.Providers) != 1 {
		t.Fatalf("acquisition discovery = %#v, want proposal-backed promoted binding", acquisition)
	}

	trace := e2etest.ReadJSONFile[caltrace.Trace](t, filepath.Join(home, "discovery", acquisition.TraceID, "trace.json"))
	if len(trace.Candidates) != 1 || trace.Candidates[0].Source != "proposal:replay" {
		t.Fatalf("trace candidates = %#v, want replay proposal candidate", trace.Candidates)
	}
	provenance := trace.Candidates[0].Provenance
	if provenance == nil || provenance.Source != "proposal:replay" || provenance.PromptVersion != "test-v1" || provenance.Model != "fixture" || provenance.SchemaVersion != "proposal.v1" || len(provenance.ProposalHash) != 64 {
		t.Fatalf("candidate provenance = %#v, want replay proposal provenance", provenance)
	}
	args, ok := trace.Candidates[0].Execution.Spec["args"].([]any)
	if !ok {
		t.Fatalf("candidate args = %#v, want JSON array", trace.Candidates[0].Execution.Spec["args"])
	}
	wantArgs := []string{"make-pdf", "--in", "{{source}}", "--out", "{{target}}"}
	if len(args) != len(wantArgs) {
		t.Fatalf("candidate args len = %d, want %d", len(args), len(wantArgs))
	}
	for index, want := range wantArgs {
		if args[index] != want {
			t.Fatalf("candidate args[%d] = %#v, want %q", index, args[index], want)
		}
	}
	if len(trace.Probes) != 1 || !trace.Probes[0].Passed || core.VerifyLevelRank(trace.Probes[0].Verify.Level) < core.VerifyLevelRank(core.VerifyLevelL2) {
		t.Fatalf("trace probes = %#v, want passing L2+ verify probe", trace.Probes)
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
	e2etest.RunJSON(t, repo, env, &runSuccess, calctlBin, "runs", "create", "--capability-id", "document.export_pdf", "--inputs-json", `{"source":`+strconv.Quote(source)+`,"target":`+strconv.Quote(target)+`}`, "--verify", "--json")
	if runSuccess.Status != "succeeded" || !runSuccess.Verified {
		t.Fatalf("run success = %#v, want verified success", runSuccess)
	}
	if len(runSuccess.Evidence) != 1 {
		t.Fatalf("run evidence = %#v, want one evidence", runSuccess.Evidence)
	}

	var metrics struct {
		Acquisition struct {
			BySource []struct {
				Source      string `json:"source"`
				Attempts    int    `json:"attempts"`
				Completed   int    `json:"completed"`
				Promotions  int    `json:"promotions"`
				ProbePasses int    `json:"probe_passes"`
			} `json:"by_source"`
		} `json:"acquisition"`
	}
	e2etest.RunJSON(t, repo, env, &metrics, calctlBin, "eval", "--json")
	if len(metrics.Acquisition.BySource) != 1 || metrics.Acquisition.BySource[0].Source != "proposal:replay" || metrics.Acquisition.BySource[0].Attempts != 1 || metrics.Acquisition.BySource[0].Completed != 1 || metrics.Acquisition.BySource[0].Promotions != 1 || metrics.Acquisition.BySource[0].ProbePasses != 1 {
		t.Fatalf("eval by source = %#v, want successful proposal replay bucket", metrics.Acquisition.BySource)
	}
}

func TestReplayProposalAcquisitionPromotesVerifySpecBinding(t *testing.T) {
	repo, calctlBin, caldBin := functionalBinaries(t)
	temp := t.TempDir()

	home := filepath.Join(temp, "home")
	providerPath := filepath.Join(temp, "proposal-exporter")
	e2etest.WriteProposalBackedExporter(t, providerPath)
	proposalPath := e2etest.WriteReplayVerifySpecProposal(t, filepath.Join(temp, "proposal-verify-spec.json"))
	env := e2etest.WithHomeEnv(os.Environ(), home)
	e2etest.StartCald(t, repo, env, caldBin)

	var acquisition struct {
		State                string                    `json:"state"`
		TraceID              string                    `json:"trace_id"`
		CapabilitiesPromoted int                       `json:"capabilities_promoted"`
		BindingsPromoted     int                       `json:"bindings_promoted"`
		Providers            []e2etest.ProviderSummary `json:"providers"`
	}
	runDiscoveryForProviderPath(t, repo, env, calctlBin, providerPath, &acquisition, "--proposal-path", proposalPath, "--json")
	if acquisition.State != "succeeded" || acquisition.CapabilitiesPromoted != 1 || acquisition.BindingsPromoted != 1 || acquisition.TraceID == "" || len(acquisition.Providers) != 1 {
		t.Fatalf("acquisition discovery = %#v, want verify-spec promoted binding", acquisition)
	}
	trace := e2etest.ReadJSONFile[caltrace.Trace](t, filepath.Join(home, "discovery", acquisition.TraceID, "trace.json"))
	if len(trace.Probes) != 1 || !trace.Probes[0].Passed || core.VerifyLevelRank(trace.Probes[0].Verify.Level) < core.VerifyLevelRank(core.VerifyLevelL2) {
		t.Fatalf("trace probes = %#v, want passing L2+ verify probe", trace.Probes)
	}
	capability := e2etest.ReadJSONFile[core.Capability](t, filepath.Join(home, "capabilities", "document.export_pdf.json"))
	if len(capability.Bindings) != 1 || capability.Bindings[0].Verify == nil || core.VerifyLevelRank(capability.Bindings[0].Verify.Level) < core.VerifyLevelRank(core.VerifyLevelL2) {
		t.Fatalf("capability = %#v, want verify-spec binding", capability)
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
	e2etest.RunJSON(t, repo, env, &runSuccess, calctlBin, "runs", "create", "--capability-id", "document.export_pdf", "--inputs-json", `{"source":`+strconv.Quote(source)+`,"target":`+strconv.Quote(target)+`}`, "--verify", "--json")
	if runSuccess.Status != "succeeded" || !runSuccess.Verified || len(runSuccess.Evidence) != 1 {
		t.Fatalf("run success = %#v, want verified verify-spec reuse", runSuccess)
	}
}

func TestControlledAcquisitionRejectsInvalidPDF(t *testing.T) {
	repo, calctlBin, caldBin := functionalBinaries(t)
	temp := t.TempDir()

	home := filepath.Join(temp, "home")
	providerPath := filepath.Join(temp, "fake-exporter")
	e2etest.WriteFakeExporter(t, providerPath, `printf '%s\n' 'not a pdf' > "$target"`)

	env := e2etest.WithHomeEnv(os.Environ(), home)
	e2etest.StartCald(t, repo, env, caldBin)

	var failure struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	runFailDiscoveryForProviderPath(t, repo, env, calctlBin, providerPath, &failure, "--mode", "rules", "--json")
	if failure.Error.Code != "verification_failed" {
		t.Fatalf("acquisition failure = %#v, want verification_failed", failure)
	}

	var capabilities struct {
		Count        int   `json:"count"`
		Capabilities []any `json:"capabilities"`
	}
	e2etest.RunJSON(t, repo, env, &capabilities, calctlBin, "capabilities", "list", "--json")
	if capabilities.Count != 0 || len(capabilities.Capabilities) != 0 {
		t.Fatalf("capability list = %#v, want no promoted capability after verification failure", capabilities)
	}

	trace := e2etest.ReadTraceByStatus(t, home, caltrace.StatusFailed)
	if trace.Error == nil || trace.Error.Code != "verification_failed" {
		t.Fatalf("failed trace error = %#v, want verification_failed", trace.Error)
	}
	if len(trace.Candidates) != 1 || len(trace.Probes) != 1 {
		t.Fatalf("failed trace = %#v, want one candidate and one probe", trace)
	}
	if trace.Probes[0].Passed || trace.Probes[0].Error == nil {
		t.Fatalf("failed trace probe = %#v, want failed probe with error", trace.Probes[0])
	}

	var metrics struct {
		Summary struct {
			Traces int `json:"traces"`
		} `json:"summary"`
		Acquisition struct {
			AttemptCount   int `json:"attempt_count"`
			FailedCount    int `json:"failed_count"`
			ProbeFailCount int `json:"probe_fail_count"`
			ByCapability   []struct {
				CapabilityID  string `json:"capability_id"`
				Attempts      int    `json:"attempts"`
				Failed        int    `json:"failed"`
				ProbeFailures int    `json:"probe_failures"`
			} `json:"by_capability"`
		} `json:"acquisition"`
	}
	e2etest.RunJSON(t, repo, env, &metrics, calctlBin, "eval", "--json")
	if metrics.Summary.Traces != 1 {
		t.Fatalf("eval summary = %#v, want failed acquisition trace", metrics.Summary)
	}
	if metrics.Acquisition.AttemptCount != 1 || metrics.Acquisition.FailedCount != 1 || metrics.Acquisition.ProbeFailCount != 1 {
		t.Fatalf("eval acquisition = %#v, want one failed acquisition attempt", metrics.Acquisition)
	}
	if len(metrics.Acquisition.ByCapability) != 1 || metrics.Acquisition.ByCapability[0].CapabilityID != "document.export_pdf" || metrics.Acquisition.ByCapability[0].Attempts != 1 || metrics.Acquisition.ByCapability[0].Failed != 1 || metrics.Acquisition.ByCapability[0].ProbeFailures != 1 {
		t.Fatalf("eval by capability = %#v, want failed document.export_pdf attempt", metrics.Acquisition.ByCapability)
	}
}
