package discovery

import (
	"testing"
	"time"

	"github.com/spacehz-lab/cal/internal/core"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestAcquisitionRunCompleteWritesTraceAndResult(t *testing.T) {
	store := newAcquisitionTestStore(t)
	now := time.Unix(0, 42).UTC()
	run := &acquisitionRun{
		store:   store,
		opts:    AcquisitionOptions{ProviderID: "provider_cli", CapabilityID: "document.convert"},
		now:     now,
		traceID: caltrace.NewID(now),
		provider: core.Provider{
			ID:   "provider_cli",
			Kind: core.ProviderKindCLI,
		},
		observations: []caltrace.Observation{{ProviderID: "provider_cli"}},
		candidates:   []caltrace.Candidate{{ProviderID: "provider_cli", CapabilityID: "document.convert"}},
		probes:       []caltrace.Probe{{CandidateIndex: 0, Passed: true}},
		promotions: []caltrace.Promotion{{
			CandidateIndex:   0,
			CapabilityAction: "created",
			BindingAction:    "created",
		}},
	}

	result, err := run.complete()
	if err != nil {
		t.Fatalf("complete() error = %v", err)
	}
	if result.State != JobStateSucceeded || result.JobID != "disc_42" || result.TraceID == "" || result.CapabilitiesPromoted != 1 || result.BindingsPromoted != 1 {
		t.Fatalf("result = %#v, want successful discovery result", result)
	}
	traces, err := store.ListTraces()
	if err != nil {
		t.Fatalf("ListTraces() error = %v", err)
	}
	if len(traces) != 1 || traces[0].Status != caltrace.StatusCompleted || len(traces[0].Promotions) != 1 {
		t.Fatalf("traces = %#v, want one completed trace with promotion", traces)
	}
}

func TestAcquisitionRunFailWritesTraceAndReturnsCodedError(t *testing.T) {
	store := newAcquisitionTestStore(t)
	run := &acquisitionRun{
		store:   store,
		opts:    AcquisitionOptions{ProviderID: "provider_cli", CapabilityID: "document.convert"},
		now:     time.Unix(0, 42).UTC(),
		traceID: caltrace.NewID(time.Unix(0, 42).UTC()),
		provider: core.Provider{
			ID:   "provider_cli",
			Kind: core.ProviderKindCLI,
		},
		observations: []caltrace.Observation{{ProviderID: "provider_cli"}},
		candidates:   []caltrace.Candidate{{ProviderID: "provider_cli", CapabilityID: "document.convert"}},
	}
	codedErr := newCodedError(CodeVerificationFailed, "verification failed")

	result, err := run.fail("verification", codedErr)
	if result.State != "" || result.TraceID != "" || result.JobID != "" {
		t.Fatalf("result = %#v, want zero result", result)
	}
	if err != codedErr {
		t.Fatalf("fail() error = %#v, want coded error", err)
	}
	traces, err := store.ListTraces()
	if err != nil {
		t.Fatalf("ListTraces() error = %v", err)
	}
	if len(traces) != 1 || traces[0].Status != caltrace.StatusFailed || traces[0].Error == nil || traces[0].Error.Code != CodeVerificationFailed {
		t.Fatalf("traces = %#v, want one failed trace with coded error", traces)
	}
}

func TestCountCapabilityCreations(t *testing.T) {
	run := &acquisitionRun{
		promotions: []caltrace.Promotion{
			{CapabilityAction: "created"},
			{CapabilityAction: "reused"},
			{CapabilityAction: "created"},
		},
	}

	got := run.countCapabilityCreations()
	if got != 2 {
		t.Fatalf("countCapabilityCreations() = %d, want 2", got)
	}
}
