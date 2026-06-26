package control

import (
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestEvalServiceComputesMetricsAndGetsTrace(t *testing.T) {
	svc := newTestService(t)
	provider := testProviderRecord()
	capability := testCapabilityRecord(t, provider.ID)
	trace := caltrace.Trace{ID: "trace_test", Status: caltrace.StatusCompleted}
	run := core.Run{
		ID:           "run_test",
		CapabilityID: capability.ID,
		Status:       core.RunStatusSucceeded,
	}
	if err := svc.store.PutProvider(provider); err != nil {
		t.Fatalf("PutProvider() error = %v", err)
	}
	if err := svc.store.PutCapability(capability); err != nil {
		t.Fatalf("PutCapability() error = %v", err)
	}
	if err := svc.store.PutTrace(trace); err != nil {
		t.Fatalf("PutTrace() error = %v", err)
	}
	if err := svc.store.PutRun(run); err != nil {
		t.Fatalf("PutRun() error = %v", err)
	}

	metrics, err := svc.Eval()
	if err != nil {
		t.Fatalf("Eval() error = %v", err)
	}
	if metrics.Summary.Providers != 1 || metrics.Summary.Capabilities != 1 || metrics.Summary.Traces != 1 || metrics.Summary.Runs != 1 {
		t.Fatalf("Eval() summary = %#v, want seeded record counts", metrics.Summary)
	}

	got, ok, err := svc.GetTrace(trace.ID)
	if err != nil {
		t.Fatalf("GetTrace() error = %v", err)
	}
	if !ok || got.ID != trace.ID {
		t.Fatalf("GetTrace() = %#v ok %v, want trace", got, ok)
	}
}
