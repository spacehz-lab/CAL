package discovery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/observe"
	"github.com/spacehz-lab/cal/internal/proposalflow"
	"github.com/spacehz-lab/cal/internal/runtime"
	calstore "github.com/spacehz-lab/cal/internal/store"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestAcquisitionRunnerRejectsMissingProvider(t *testing.T) {
	store := newAcquisitionTestStore(t)
	_, err := acquisitionTestRunner("document.export_pdf").Run(context.Background(), store, AcquisitionOptions{
		ProviderID:   "provider_missing",
		CapabilityID: "document.export_pdf",
	})
	assertCodedError(t, err, "provider_not_found")
}

func TestAcquisitionRunnerRejectsAppProvider(t *testing.T) {
	store := newAcquisitionTestStore(t)
	if err := store.PutProvider(core.Provider{
		ID:   "provider_app",
		Name: "Fake App",
		Kind: core.ProviderKindApp,
		Path: "/Applications/Fake.app",
	}); err != nil {
		t.Fatalf("PutProvider() error = %v", err)
	}

	_, err := acquisitionTestRunner("document.export_pdf").Run(context.Background(), store, AcquisitionOptions{
		ProviderID:   "provider_app",
		CapabilityID: "document.export_pdf",
	})
	assertCodedError(t, err, "unsupported_provider_kind")
}

func TestAcquisitionRunnerReportsMissingCandidate(t *testing.T) {
	store := newAcquisitionTestStore(t)
	if err := putCLIProvider(t, store, writeAcquisitionScript(t, true)); err != nil {
		t.Fatalf("put provider: %v", err)
	}

	_, err := acquisitionTestRunner("").Run(context.Background(), store, AcquisitionOptions{
		ProviderID:   "provider_cli",
		CapabilityID: "document.export_pdf",
	})
	assertCodedError(t, err, "candidate_not_found")
	trace := assertSingleFailedTrace(t, store, "candidate_not_found")
	if len(trace.Observations) != 1 || len(trace.Candidates) != 0 || len(trace.Probes) != 0 {
		t.Fatalf("trace = %#v, want observation without candidates or probes", trace)
	}
}

func TestAcquisitionRunnerReportsVerificationFailure(t *testing.T) {
	store := newAcquisitionTestStore(t)
	if err := putCLIProvider(t, store, writeAcquisitionScript(t, false)); err != nil {
		t.Fatalf("put provider: %v", err)
	}

	_, err := acquisitionTestRunner("document.export_pdf").Run(context.Background(), store, AcquisitionOptions{
		ProviderID:   "provider_cli",
		CapabilityID: "document.export_pdf",
	})
	assertCodedError(t, err, "verification_failed")
	trace := assertSingleFailedTrace(t, store, "verification_failed")
	if len(trace.Observations) != 1 || len(trace.Candidates) != 1 || len(trace.Probes) != 1 {
		t.Fatalf("trace = %#v, want observation, candidate, and failed probe", trace)
	}
	if trace.Probes[0].Passed || trace.Probes[0].Error == nil {
		t.Fatalf("trace probe = %#v, want failed probe with error", trace.Probes[0])
	}
	if trace.Probes[0].Inputs["target"] == nil {
		t.Fatalf("trace probe inputs = %#v, want failed probe inputs", trace.Probes[0].Inputs)
	}
	if _, ok, err := store.GetCapability("document.export_pdf"); err != nil || ok {
		t.Fatalf("GetCapability() = _, %v, %v, want no promoted capability", ok, err)
	}
}

func TestAcquisitionRunnerPromotesBindingAndTrace(t *testing.T) {
	store := newAcquisitionTestStore(t)
	if err := putCLIProvider(t, store, writeAcquisitionScript(t, true)); err != nil {
		t.Fatalf("put provider: %v", err)
	}

	result, err := acquisitionTestRunner("document.export_pdf").Run(context.Background(), store, AcquisitionOptions{
		ProviderID:   "provider_cli",
		CapabilityID: "document.export_pdf",
	})
	if err != nil {
		t.Fatalf("AcquisitionRunner.Run() error = %v", err)
	}
	if result.State != JobStateSucceeded || result.CapabilitiesPromoted != 1 || result.BindingsPromoted != 1 || result.TraceID == "" {
		t.Fatalf("result = %#v, want promoted capability and trace", result)
	}

	capability, ok, err := store.GetCapability("document.export_pdf")
	if err != nil {
		t.Fatalf("GetCapability() error = %v", err)
	}
	if !ok || len(capability.Bindings) != 1 || capability.Bindings[0].ProviderID != "provider_cli" {
		t.Fatalf("capability = %#v, %v, want promoted provider_cli binding", capability, ok)
	}
	trace, ok, err := store.GetTrace(result.TraceID)
	if err != nil {
		t.Fatalf("GetTrace() error = %v", err)
	}
	promotions := promotionSummaries(trace)
	if !ok || len(promotions) != 1 || promotions[0].BindingID == "" || len(trace.Candidates) != 1 || len(trace.Probes) != 1 {
		t.Fatalf("trace = %#v, %v, want promotion, candidate, and probe", trace, ok)
	}
	if promotions[0].CapabilityAction != "created" || promotions[0].BindingAction != "created" {
		t.Fatalf("trace promotion = %#v, want created capability and binding", promotions[0])
	}
	if trace.Probes[0].Inputs["target"] == nil {
		t.Fatalf("trace probe inputs = %#v, want promoted probe inputs", trace.Probes[0].Inputs)
	}
}

func TestAcquisitionRunnerReportsProposalDuration(t *testing.T) {
	store := newAcquisitionTestStore(t)
	if err := putCLIProvider(t, store, writeAcquisitionScript(t, true)); err != nil {
		t.Fatalf("put provider: %v", err)
	}
	runner := NewAcquisitionRunner(fakeObserver{}, slowProposer{
		capabilityID: "document.export_pdf",
		delay:        2 * time.Millisecond,
	})

	result, err := runner.Run(context.Background(), store, AcquisitionOptions{
		ProviderID:   "provider_cli",
		CapabilityID: "document.export_pdf",
	})
	if err != nil {
		t.Fatalf("AcquisitionRunner.Run() error = %v", err)
	}
	if result.ProposalDurationMS <= 0 {
		t.Fatalf("proposal duration = %d, want structured proposal duration", result.ProposalDurationMS)
	}
}

func TestAcquisitionRunnerMergesExistingCapability(t *testing.T) {
	store := newAcquisitionTestStore(t)
	if err := putCLIProvider(t, store, writeAcquisitionScript(t, true)); err != nil {
		t.Fatalf("put provider: %v", err)
	}
	runner := acquisitionTestRunner("document.export_pdf")
	if _, err := runner.Run(context.Background(), store, AcquisitionOptions{ProviderID: "provider_cli", CapabilityID: "document.export_pdf"}); err != nil {
		t.Fatalf("first AcquisitionRunner.Run() error = %v", err)
	}

	result, err := runner.Run(context.Background(), store, AcquisitionOptions{ProviderID: "provider_cli", CapabilityID: "document.export_pdf"})
	if err != nil {
		t.Fatalf("second AcquisitionRunner.Run() error = %v", err)
	}
	if result.CapabilitiesPromoted != 0 || result.BindingsPromoted != 1 {
		t.Fatalf("second result = %#v, want existing capability merge", result)
	}
	capability, ok, err := store.GetCapability("document.export_pdf")
	if err != nil {
		t.Fatalf("GetCapability() error = %v", err)
	}
	if !ok || len(capability.Bindings) != 1 {
		t.Fatalf("capability = %#v, %v, want one replacement binding", capability, ok)
	}
	trace, ok, err := store.GetTrace(result.TraceID)
	if err != nil {
		t.Fatalf("GetTrace() error = %v", err)
	}
	promotions := promotionSummaries(trace)
	if !ok || len(promotions) != 1 || promotions[0].CapabilityAction != "reused" || promotions[0].BindingAction != "updated" {
		t.Fatalf("trace = %#v, %v, want reused capability and updated binding", trace, ok)
	}
}

func TestAcquisitionRunnerRecordsReusedCapabilityWithCreatedBinding(t *testing.T) {
	store := newAcquisitionTestStore(t)
	if err := putCLIProvider(t, store, writeAcquisitionScript(t, true)); err != nil {
		t.Fatalf("put provider: %v", err)
	}
	runner := acquisitionTestRunner("document.export_pdf")
	if _, err := runner.Run(context.Background(), store, AcquisitionOptions{ProviderID: "provider_cli", CapabilityID: "document.export_pdf"}); err != nil {
		t.Fatalf("first AcquisitionRunner.Run() error = %v", err)
	}
	if err := store.PutProvider(core.Provider{
		ID:   "provider_other",
		Name: "fake-other",
		Kind: core.ProviderKindCLI,
		Path: writeAcquisitionScript(t, true),
	}); err != nil {
		t.Fatalf("PutProvider(other) error = %v", err)
	}

	result, err := runner.Run(context.Background(), store, AcquisitionOptions{ProviderID: "provider_other", CapabilityID: "document.export_pdf"})
	if err != nil {
		t.Fatalf("second provider AcquisitionRunner.Run() error = %v", err)
	}
	trace, ok, err := store.GetTrace(result.TraceID)
	if err != nil {
		t.Fatalf("GetTrace() error = %v", err)
	}
	promotions := promotionSummaries(trace)
	if !ok || len(promotions) != 1 || promotions[0].CapabilityAction != "reused" || promotions[0].BindingAction != "created" {
		t.Fatalf("trace = %#v, %v, want reused capability and created binding", trace, ok)
	}
}

func TestAcquisitionRunnerPromotesMultipleCandidates(t *testing.T) {
	store := newAcquisitionTestStore(t)
	if err := putCLIProvider(t, store, writeAcquisitionScript(t, true)); err != nil {
		t.Fatalf("put provider: %v", err)
	}
	runner := NewAcquisitionRunner(fakeObserver{}, multiProposer{capabilityIDs: []string{"document.export_pdf", "image.resize"}})

	result, err := runner.Run(context.Background(), store, AcquisitionOptions{ProviderID: "provider_cli"})
	if err != nil {
		t.Fatalf("AcquisitionRunner.Run() error = %v", err)
	}
	if result.State != JobStateSucceeded || result.CapabilitiesPromoted != 2 || result.BindingsPromoted != 2 {
		t.Fatalf("result = %#v, want two promoted capabilities and bindings", result)
	}
	for _, capabilityID := range []string{"document.export_pdf", "image.resize"} {
		capability, ok, err := store.GetCapability(capabilityID)
		if err != nil || !ok || len(capability.Bindings) != 1 {
			t.Fatalf("GetCapability(%s) = %#v, %v, %v, want one binding", capabilityID, capability, ok, err)
		}
	}
	trace, ok, err := store.GetTrace(result.TraceID)
	if err != nil {
		t.Fatalf("GetTrace() error = %v", err)
	}
	if !ok || len(trace.Candidates) != 2 || len(trace.Probes) != 2 || len(promotionSummaries(trace)) != 2 {
		t.Fatalf("trace = %#v, %v, want two candidates, probes, and promotions", trace, ok)
	}
	for index, probe := range trace.Probes {
		if probe.CandidateIndex != index || !probe.Passed {
			t.Fatalf("probe[%d] = %#v, want passed probe with matching candidate index", index, probe)
		}
	}
	for index, promotion := range promotionSummaries(trace) {
		if promotion.CandidateIndex != index {
			t.Fatalf("promotion[%d] = %#v, want matching candidate index", index, promotion)
		}
	}
}

func TestAcquisitionRunnerSucceedsWithPartialCandidateFailure(t *testing.T) {
	store := newAcquisitionTestStore(t)
	if err := putCLIProvider(t, store, writeAcquisitionScript(t, true)); err != nil {
		t.Fatalf("put provider: %v", err)
	}
	runner := NewAcquisitionRunner(fakeObserver{}, multiProposer{
		capabilityIDs: []string{"document.export_pdf", "image.resize"},
		badIndexes:    map[int]struct{}{1: {}},
	})

	result, err := runner.Run(context.Background(), store, AcquisitionOptions{ProviderID: "provider_cli"})
	if err != nil {
		t.Fatalf("AcquisitionRunner.Run() error = %v", err)
	}
	if result.State != JobStateSucceeded || result.CapabilitiesPromoted != 1 || result.BindingsPromoted != 1 {
		t.Fatalf("result = %#v, want one promoted candidate", result)
	}
	trace, ok, err := store.GetTrace(result.TraceID)
	if err != nil {
		t.Fatalf("GetTrace() error = %v", err)
	}
	if !ok || len(trace.Candidates) != 2 || len(trace.Probes) != 2 || len(promotionSummaries(trace)) != 1 {
		t.Fatalf("trace = %#v, %v, want two probes and one promotion", trace, ok)
	}
	if !trace.Probes[0].Passed || trace.Probes[1].Passed || trace.Probes[1].Error == nil {
		t.Fatalf("trace probes = %#v, want first passed and second failed", trace.Probes)
	}
}

func TestAcquisitionRunnerFailsWhenAllCandidatesFail(t *testing.T) {
	store := newAcquisitionTestStore(t)
	if err := putCLIProvider(t, store, writeAcquisitionScript(t, true)); err != nil {
		t.Fatalf("put provider: %v", err)
	}
	runner := NewAcquisitionRunner(fakeObserver{}, multiProposer{
		capabilityIDs: []string{"document.export_pdf", "image.resize"},
		badIndexes:    map[int]struct{}{0: {}, 1: {}},
	})

	_, err := runner.Run(context.Background(), store, AcquisitionOptions{ProviderID: "provider_cli", CapabilityID: "document.export_pdf"})
	assertCodedError(t, err, "verification_failed")
	trace := assertSingleFailedTrace(t, store, "verification_failed")
	if len(trace.Candidates) != 2 || len(trace.Probes) != 2 || len(promotionSummaries(trace)) != 0 {
		t.Fatalf("trace = %#v, want failed candidates without promotion", trace)
	}
	for _, probe := range trace.Probes {
		if probe.Passed || probe.Error == nil {
			t.Fatalf("probe = %#v, want failed probe", probe)
		}
	}
}

func TestAcquisitionRunnerPassesCatalogToProposer(t *testing.T) {
	store := newAcquisitionTestStore(t)
	if err := putCLIProvider(t, store, writeAcquisitionScript(t, true)); err != nil {
		t.Fatalf("put provider: %v", err)
	}
	for _, capability := range []core.Capability{
		{ID: "image.resize"},
		{ID: "document.export_pdf"},
	} {
		if err := store.PutCapability(capability); err != nil {
			t.Fatalf("PutCapability() error = %v", err)
		}
	}
	proposer := &capturingProposer{}
	runner := NewAcquisitionRunner(fakeObserver{}, proposer)

	_, err := runner.Run(context.Background(), store, AcquisitionOptions{ProviderID: "provider_cli", CapabilityID: "image.resize"})
	assertCodedError(t, err, "candidate_not_found")
	assertIDs(t, capabilityIDs(proposer.catalog), []string{"document.export_pdf", "image.resize"})
	if proposer.debugFilter != "image.resize" {
		t.Fatalf("debug filter = %q, want image.resize", proposer.debugFilter)
	}
}

func TestAcquisitionRunnerUsesProposalProbePlan(t *testing.T) {
	store := newAcquisitionTestStore(t)
	if err := putCLIProvider(t, store, writeAcquisitionScript(t, true)); err != nil {
		t.Fatalf("put provider: %v", err)
	}
	runner := NewAcquisitionRunner(fakeObserver{}, fakeProposer{capabilityID: "custom.cap"})

	result, err := runner.Run(context.Background(), store, AcquisitionOptions{ProviderID: "provider_cli"})
	if err != nil {
		t.Fatalf("AcquisitionRunner.Run() error = %v", err)
	}
	if result.State != JobStateSucceeded || result.CapabilitiesPromoted != 1 {
		t.Fatalf("result = %#v, want successful custom capability promotion", result)
	}
	if _, ok, err := store.GetCapability("custom.cap"); err != nil || !ok {
		t.Fatalf("GetCapability(custom.cap) = _, %v, %v, want promoted capability", ok, err)
	}
}

func acquisitionTestRunner(capabilityID string) AcquisitionRunner {
	return NewAcquisitionRunner(fakeObserver{}, fakeProposer{capabilityID: capabilityID})
}

func TestNewAcquisitionRunnerConfiguresComponents(t *testing.T) {
	runner := NewAcquisitionRunner(fakeObserver{}, fakeProposer{capabilityID: "document.export_pdf"})
	if runner.observer == nil || runner.proposer == nil {
		t.Fatalf("NewAcquisitionRunner() = %#v, want configured observer and proposer", runner)
	}
}

func TestAcquisitionRunnerRejectsMissingProposer(t *testing.T) {
	store := newAcquisitionTestStore(t)
	runner := NewAcquisitionRunner(fakeObserver{}, nil)
	_, err := runner.Run(context.Background(), store, AcquisitionOptions{ProviderID: "provider_cli"})
	assertCodedError(t, err, "proposer_unavailable")
}

type fakeObserver struct{}

func (fakeObserver) Observe(context.Context, core.Provider) (observe.Result, error) {
	return observe.Result{
		Observations: []observe.Observation{{
			Type:   "cli_output",
			Source: "test",
			Content: map[string]any{
				"text": "fake help",
			},
		}},
	}, nil
}

type fakeProposer struct {
	capabilityID string
}

func (proposer fakeProposer) Propose(_ context.Context, request proposalflow.Request) (proposalflow.Result, error) {
	if proposer.capabilityID == "" {
		return proposalflow.Result{}, nil
	}
	return proposalResult(request.Provider.ID, []caltrace.Candidate{{
		ProviderID:   request.Provider.ID,
		CapabilityID: proposer.capabilityID,
		Description:  testCapabilityDescription(proposer.capabilityID),
		Execution: core.Execution{
			Kind: core.ExecutionKindCLI,
			Spec: map[string]any{
				"args": []string{"export-pdf", "--target", "{{target}}"},
			},
		},
	}}), nil
}

type slowProposer struct {
	capabilityID string
	delay        time.Duration
}

func (proposer slowProposer) Propose(ctx context.Context, request proposalflow.Request) (proposalflow.Result, error) {
	timer := time.NewTimer(proposer.delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return proposalflow.Result{}, ctx.Err()
	case <-timer.C:
		return fakeProposer{capabilityID: proposer.capabilityID}.Propose(ctx, request)
	}
}

type multiProposer struct {
	capabilityIDs []string
	badIndexes    map[int]struct{}
}

func (proposer multiProposer) Propose(_ context.Context, request proposalflow.Request) (proposalflow.Result, error) {
	candidates := make([]caltrace.Candidate, 0, len(proposer.capabilityIDs))
	for index, capabilityID := range proposer.capabilityIDs {
		args := []string{"export-pdf", "--target", "{{target}}"}
		if _, bad := proposer.badIndexes[index]; bad {
			args = []string{"missing-command"}
		}
		candidates = append(candidates, caltrace.Candidate{
			ProviderID:   request.Provider.ID,
			CapabilityID: capabilityID,
			Description:  testCapabilityDescription(capabilityID),
			Execution: core.Execution{
				Kind: core.ExecutionKindCLI,
				Spec: map[string]any{
					"args": args,
				},
			},
		})
	}
	return proposalResult(request.Provider.ID, candidates), nil
}

func testCapabilityDescription(capabilityID string) string {
	switch capabilityID {
	case "document.export_pdf":
		return "Export a document to a PDF artifact."
	case "image.resize":
		return "Resize an image artifact."
	default:
		return "Execute " + capabilityID + "."
	}
}

type capturingProposer struct {
	catalog     []core.Capability
	debugFilter string
}

func (proposer *capturingProposer) Propose(_ context.Context, request proposalflow.Request) (proposalflow.Result, error) {
	proposer.catalog = append([]core.Capability(nil), request.Catalog...)
	proposer.debugFilter = request.DebugFilter
	return proposalflow.Result{}, nil
}

func proposalResult(providerID string, candidates []caltrace.Candidate) proposalflow.Result {
	probePlans := make([]proposalflow.ProbePlan, 0, len(candidates))
	for index := range candidates {
		if candidates[index].ProviderID == "" {
			candidates[index].ProviderID = providerID
		}
		probePlans = append(probePlans, proposalflow.ProbePlan{
			CandidateIndex: index,
			Inputs:         map[string]any{"target": "{{workdir}}/output.any"},
			Verifier:       core.Verifier{ID: "file_exists"},
		})
	}
	return proposalflow.Result{
		Candidates: candidates,
		ProbePlans: probePlans,
	}
}

func newAcquisitionTestStore(t *testing.T) *calstore.Store {
	t.Helper()
	home := t.TempDir()
	t.Setenv("CAL_HOME", home)
	store, err := calstore.Open(home)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := store.Ensure(); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	installAcquisitionRunnerTestVerifier(t)
	return store
}

func installAcquisitionRunnerTestVerifier(t *testing.T) {
	t.Helper()
	err := runtime.InstallVerifier(runtime.GeneratedVerifierPackage{
		ID: "file_exists",
		VerifyPY: `import json
import os
import sys

request = json.load(sys.stdin)
verifier_id = request["verifier"]["id"]
target = (request.get("inputs") or {}).get("target")
if not isinstance(target, str) or not os.path.exists(target):
    print(json.dumps({"passed": False, "error": {"code": "file_missing", "message": "target file is missing"}}))
    sys.exit(0)
print(json.dumps({
    "passed": True,
    "evidence": [{"id": verifier_id, "type": verifier_id, "content": {"target": target}}],
    "outputs": {"target": target},
}))
`,
	})
	if err != nil {
		t.Fatalf("InstallVerifier(file_exists) error = %v", err)
	}
}

func putCLIProvider(t *testing.T, store *calstore.Store, path string) error {
	t.Helper()
	return store.PutProvider(core.Provider{
		ID:   "provider_cli",
		Name: "fake",
		Kind: core.ProviderKindCLI,
		Path: path,
	})
}

func writeAcquisitionScript(t *testing.T, createsTarget bool) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake-cli")
	script := "#!/bin/sh\n"
	if createsTarget {
		script += `if [ "$1" = "export-pdf" ]; then
  while [ "$#" -gt 0 ]; do
    if [ "$1" = "--target" ]; then
      printf '%s\n' '%PDF-1.4' '1 0 obj' '<< /Type /Catalog /Pages 2 0 R >>' 'endobj' '2 0 obj' '<< /Type /Pages /Kids [3 0 R] /Count 1 >>' 'endobj' '3 0 obj' '<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] /Contents 4 0 R >>' 'endobj' '4 0 obj' '<< /Length 44 >>' 'stream' 'BT /F1 12 Tf 10 100 Td (fake pdf) Tj ET' 'endstream' 'endobj' 'xref' '0 5' '0000000000 65535 f ' '0000000009 00000 n ' '0000000058 00000 n ' '0000000115 00000 n ' '0000000202 00000 n ' 'trailer' '<< /Root 1 0 R /Size 5 >>' 'startxref' '295' '%%EOF' > "$2"
      exit 0
    fi
    shift
  done
fi
exit 64
`
	} else {
		script += "exit 64\n"
	}
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write acquisition script: %v", err)
	}
	return path
}

func assertCodedError(t *testing.T, err error, code string) {
	t.Helper()
	codedErr, ok := err.(CodedError)
	if !ok || codedErr.Code != code {
		t.Fatalf("error = %#v, want %s", err, code)
	}
	if fmt.Sprint(err) == "" {
		t.Fatal("error string is empty")
	}
}

func assertIDs(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("ids = %#v, want %#v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("ids = %#v, want %#v", got, want)
		}
	}
}

func capabilityIDs(capabilities []core.Capability) []string {
	ids := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		ids = append(ids, capability.ID)
	}
	return ids
}

func assertSingleFailedTrace(t *testing.T, store *calstore.Store, code string) caltrace.Trace {
	t.Helper()

	traces, err := store.ListTraces()
	if err != nil {
		t.Fatalf("ListTraces() error = %v", err)
	}
	if len(traces) != 1 {
		t.Fatalf("traces = %#v, want one failed trace", traces)
	}
	trace := traces[0]
	if trace.Status != caltrace.StatusFailed || trace.Error == nil || trace.Error.Code != code {
		t.Fatalf("trace = %#v, want failed trace with error %s", trace, code)
	}
	if trace.Hint != "document.export_pdf" || len(trace.ProviderIDs) != 1 || trace.ProviderIDs[0] != "provider_cli" {
		t.Fatalf("trace target = %#v, %#v, want document.export_pdf/provider_cli", trace.Hint, trace.ProviderIDs)
	}
	return trace
}

func promotionSummaries(trace caltrace.Trace) []caltrace.Promotion {
	return trace.Promotions
}
