package eval

import (
	"errors"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/store"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestComputeCountsStoreRecords(t *testing.T) {
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := s.Ensure(); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if err := s.PutProvider(core.Provider{
		ID:   "provider_fake",
		Kind: core.ProviderKindCLI,
		Path: "/tmp/fake",
	}); err != nil {
		t.Fatalf("PutProvider() error = %v", err)
	}
	if err := s.PutCapability(core.Capability{
		ID:          "document.export_pdf",
		Description: "Export a document to a PDF artifact.",
		Bindings: []core.Binding{
			{
				ID:           "binding_promoted",
				CapabilityID: "document.export_pdf",
				ProviderID:   "provider_fake",
				Execution:    core.Execution{Kind: core.ExecutionKindCLI},
				Verify:       evalVerifySpecPtr(core.VerifyLevelL2),
				Evidence:     []core.EvidenceRef{{ID: "evidence_fake"}},
				State:        core.BindingStatePromoted,
			},
		},
	}); err != nil {
		t.Fatalf("PutCapability() error = %v", err)
	}
	if err := s.PutTrace(caltrace.Trace{
		ID:     "trace_fake",
		Status: caltrace.StatusCompleted,
		Hint:   "document.export_pdf",
		Candidates: []caltrace.Candidate{{
			ProviderID:   "provider_fake",
			CapabilityID: "document.export_pdf",
			Description:  "Export a document to a PDF artifact.",
			Source:       "proposal:replay",
			Provenance: &caltrace.CandidateProvenance{
				Source:        "proposal:replay",
				PromptVersion: "prompt-v1",
				Model:         "fixture-model",
				SchemaVersion: "proposal.v1",
				ProposalHash:  "abc123",
			},
			Execution: core.Execution{Kind: core.ExecutionKindCLI},
		}},
		Probes: []caltrace.Probe{{
			CandidateIndex: 0,
			Passed:         true,
			Verify:         evalVerifySpec(core.VerifyLevelL2),
			Evidence:       []core.EvidenceRef{{ID: "evidence_fake"}},
		}},
		Promotions: []caltrace.Promotion{{
			CandidateIndex:   0,
			CapabilityID:     "document.export_pdf",
			BindingID:        "binding_promoted",
			ProviderID:       "provider_fake",
			CapabilityAction: "created",
			BindingAction:    "created",
		}},
	}); err != nil {
		t.Fatalf("PutTrace() error = %v", err)
	}
	if err := s.PutTrace(caltrace.Trace{
		ID:     "trace_reused",
		Status: caltrace.StatusCompleted,
		Candidates: []caltrace.Candidate{{
			ProviderID:   "provider_fake",
			CapabilityID: "document.export_pdf",
			Description:  "Export a document to a PDF artifact.",
			Source:       "proposal:replay",
			Execution:    core.Execution{Kind: core.ExecutionKindCLI},
		}},
		Probes: []caltrace.Probe{{
			CandidateIndex: 0,
			Passed:         true,
			Verify:         evalVerifySpec(core.VerifyLevelL2),
			Evidence:       []core.EvidenceRef{{ID: "evidence_reused"}},
		}},
		Promotions: []caltrace.Promotion{{
			CandidateIndex:   0,
			CapabilityID:     "document.export_pdf",
			BindingID:        "binding_promoted",
			ProviderID:       "provider_fake",
			CapabilityAction: "reused",
			BindingAction:    "updated",
		}},
	}); err != nil {
		t.Fatalf("PutTrace(reused) error = %v", err)
	}
	if err := s.PutTrace(caltrace.Trace{
		ID:     "trace_failed",
		Status: caltrace.StatusFailed,
		Hint:   "document.export_pdf",
		Candidates: []caltrace.Candidate{{
			ProviderID:   "provider_fake",
			CapabilityID: "document.export_pdf",
			Description:  "Export a document to a PDF artifact.",
			Source:       "rules:test",
			Execution:    core.Execution{Kind: core.ExecutionKindCLI},
		}},
		Probes: []caltrace.Probe{{
			CandidateIndex: 0,
			Passed:         false,
			Verify:         evalVerifySpec(core.VerifyLevelL2),
			Error:          &core.RecordError{Code: "verification_failed", Message: "invalid pdf"},
		}},
		Error: &core.RecordError{Code: "verification_failed", Message: "invalid pdf"},
	}); err != nil {
		t.Fatalf("PutTrace(failed) error = %v", err)
	}
	if err := s.PutRun(core.Run{
		ID:           "run_success",
		CapabilityID: "document.export_pdf",
		BindingID:    "binding_promoted",
		ProviderID:   "provider_fake",
		Status:       core.RunStatusSucceeded,
		Verified:     true,
		DurationMS:   40,
	}); err != nil {
		t.Fatalf("PutRun(success) error = %v", err)
	}
	if err := s.PutRun(core.Run{
		ID:           "run_failed",
		CapabilityID: "document.export_pdf",
		BindingID:    "binding_promoted",
		ProviderID:   "provider_fake",
		Status:       core.RunStatusFailed,
		DurationMS:   20,
	}); err != nil {
		t.Fatalf("PutRun(failed) error = %v", err)
	}
	if err := s.PutRun(core.Run{
		ID:           "run_verified_failed",
		CapabilityID: "document.export_pdf",
		BindingID:    "binding_promoted",
		ProviderID:   "provider_fake",
		Status:       core.RunStatusFailed,
		DurationMS:   30,
		Error:        &core.RecordError{Code: "verification_failed", Message: "invalid pdf"},
	}); err != nil {
		t.Fatalf("PutRun(verified failed) error = %v", err)
	}

	metrics, err := Compute(s)
	if err != nil {
		t.Fatalf("Compute() error = %v", err)
	}
	if metrics.Summary.Providers != 1 || metrics.Summary.Capabilities != 1 || metrics.Summary.Bindings != 1 || metrics.Summary.PromotedBindings != 1 || metrics.Summary.Traces != 3 || metrics.Summary.Runs != 3 {
		t.Fatalf("metrics = %#v, want provider/capability/binding/trace counts", metrics)
	}
	if metrics.Acquisition.AttemptCount != 3 || metrics.Acquisition.CompletedCount != 2 || metrics.Acquisition.FailedCount != 1 {
		t.Fatalf("acquisition attempts = %#v, want two completed and one failed attempt", metrics.Acquisition)
	}
	if metrics.Acquisition.CandidateCount != 3 || metrics.Acquisition.ProbeCount != 3 || metrics.Acquisition.ProbePassCount != 2 || metrics.Acquisition.ProbeFailCount != 1 {
		t.Fatalf("acquisition probes = %#v, want two passed and one failed probe", metrics.Acquisition)
	}
	if metrics.Acquisition.PromotionCount != 2 || metrics.Acquisition.BindingPromotionRate != 2.0/3.0 || metrics.Acquisition.ProbeSuccessRate != 2.0/3.0 {
		t.Fatalf("acquisition rates = %#v, want two thirds promotion and probe success rates", metrics.Acquisition)
	}
	if metrics.Acquisition.CapabilityCreatedCount != 1 || metrics.Acquisition.CapabilityReusedCount != 1 || metrics.Acquisition.BindingCreatedCount != 1 || metrics.Acquisition.BindingUpdatedCount != 1 {
		t.Fatalf("promotion action counts = %#v, want created/reused and created/updated counts", metrics.Acquisition)
	}
	if len(metrics.Acquisition.ByCapability) != 1 {
		t.Fatalf("by capability = %#v, want one capability bucket", metrics.Acquisition.ByCapability)
	}
	byCapability := metrics.Acquisition.ByCapability[0]
	if byCapability.CapabilityID != "document.export_pdf" || byCapability.Attempts != 3 || byCapability.Completed != 2 || byCapability.Failed != 1 || byCapability.Promotions != 2 || byCapability.Candidates != 3 || byCapability.Probes != 3 || byCapability.ProbePasses != 2 || byCapability.ProbeFailures != 1 {
		t.Fatalf("by capability = %#v, want document.export_pdf acquisition counts", byCapability)
	}
	if len(metrics.Acquisition.BySource) != 2 {
		t.Fatalf("by source = %#v, want proposal and rules buckets", metrics.Acquisition.BySource)
	}
	proposalSource := metrics.Acquisition.BySource[0]
	rulesSource := metrics.Acquisition.BySource[1]
	if proposalSource.Source != "proposal:replay" || proposalSource.Attempts != 2 || proposalSource.Completed != 2 || proposalSource.Promotions != 2 || proposalSource.ProbePasses != 2 {
		t.Fatalf("proposal source = %#v, want successful proposal bucket", proposalSource)
	}
	if rulesSource.Source != "rules:test" || rulesSource.Attempts != 1 || rulesSource.Failed != 1 || rulesSource.ProbeFailures != 1 {
		t.Fatalf("rules source = %#v, want failed rules bucket", rulesSource)
	}
	if metrics.Reuse.RunCount != 3 || metrics.Reuse.RunSuccessCount != 1 || metrics.Reuse.RunFailureCount != 2 || metrics.Reuse.VerifiedRunCount != 2 || metrics.Reuse.VerifierFailCount != 1 {
		t.Fatalf("reuse counts = %#v, want run and verified run counts", metrics.Reuse)
	}
	if metrics.Reuse.RunSuccessRate != 1.0/3.0 || metrics.Reuse.VerifiedSuccessRate != 0.5 || metrics.Reuse.VerifierFailureRate != 0.5 || metrics.Reuse.AvgRunDurationMS != 30 {
		t.Fatalf("reuse rates = %#v, want run, verified, verifier, and duration rates", metrics.Reuse)
	}
}

func TestEvaluatorComputeReturnsLoadErrors(t *testing.T) {
	for _, stage := range []string{"providers", "capabilities", "runs", "traces"} {
		t.Run(stage, func(t *testing.T) {
			_, err := NewEvaluator(failingEvalStore{stage: stage}).Compute()
			if err == nil {
				t.Fatalf("Compute() error = nil, want %s load error", stage)
			}
		})
	}
}

type failingEvalStore struct {
	stage string
}

func (store failingEvalStore) ListProviders() ([]core.Provider, error) {
	if store.stage == "providers" {
		return nil, errors.New("providers failed")
	}
	return []core.Provider{}, nil
}

func (store failingEvalStore) ListCapabilities() ([]core.Capability, error) {
	if store.stage == "capabilities" {
		return nil, errors.New("capabilities failed")
	}
	return []core.Capability{}, nil
}

func (store failingEvalStore) ListRuns() ([]core.Run, error) {
	if store.stage == "runs" {
		return nil, errors.New("runs failed")
	}
	return []core.Run{}, nil
}

func (store failingEvalStore) ListTraces() ([]caltrace.Trace, error) {
	if store.stage == "traces" {
		return nil, errors.New("traces failed")
	}
	return []caltrace.Trace{}, nil
}

func evalVerifySpecPtr(level core.VerifyLevel) *core.VerifySpec {
	verify := evalVerifySpec(level)
	return &verify
}

func evalVerifySpec(level core.VerifyLevel) core.VerifySpec {
	return core.VerifySpec{
		Level:  level,
		Method: core.VerifyMethodExecute,
		Checks: []core.VerifyCheck{{Subject: "target", Predicate: core.VerifyPredicateExists}},
	}
}
