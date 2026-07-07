package tracelog

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestWriterStartWritesRunningTrace(t *testing.T) {
	store := newFakeStore()
	writer := NewWriter(store, fixedNow)

	result, err := writer.Start(context.Background(), &Request{Hint: "discover pdf"})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if result.Trace.ID != "trace_123000000456" || result.Trace.Status != model.TraceStatusRunning {
		t.Fatalf("trace = %#v, want generated running trace", result.Trace)
	}
	if result.Trace.StartedAt != fixedNow().UTC().Format(time.RFC3339Nano) || result.Trace.EndedAt != "" {
		t.Fatalf("trace times = %#v, want started only", result.Trace)
	}
	if len(store.traces) != 1 || store.traces[0].ID != result.Trace.ID {
		t.Fatalf("stored traces = %#v, want running trace", store.traces)
	}
}

func TestWriterCompleteWritesStageResults(t *testing.T) {
	store := newFakeStore()
	writer := NewWriter(store, fixedNow)
	req := requestWithResults()

	result, err := writer.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	trace := result.Trace
	if trace.Status != model.TraceStatusCompleted || trace.EndedAt == "" {
		t.Fatalf("trace = %#v, want completed with ended time", trace)
	}
	if len(trace.ProviderIDs) != 1 || len(trace.Observations) != 1 || len(trace.Candidates) != 1 || len(trace.Probes) != 1 || len(trace.Promotions) != 1 {
		t.Fatalf("trace = %#v, want all stage results", trace)
	}
	if trace.Proposal == nil || trace.Proposal.Model != "gpt-test" {
		t.Fatalf("proposal = %#v, want proposal diagnostics", trace.Proposal)
	}
}

func TestWriterFailWritesPartialResultsAndError(t *testing.T) {
	store := newFakeStore()
	writer := NewWriter(store, fixedNow)
	req := requestWithResults()
	req.Probes = nil
	req.Promotions = nil
	req.Error = &model.RecordError{Code: "probe_failed", Message: "probe failed"}

	result, err := writer.Fail(context.Background(), req)
	if err != nil {
		t.Fatalf("Fail() error = %v", err)
	}
	if result.Trace.Status != model.TraceStatusFailed || result.Trace.Error == nil || result.Trace.Error.Code != "probe_failed" {
		t.Fatalf("trace = %#v, want failed trace with error", result.Trace)
	}
	if len(result.Trace.Candidates) != 1 || len(result.Trace.Probes) != 0 {
		t.Fatalf("trace = %#v, want partial results", result.Trace)
	}
}

func TestWriterCancelWritesCanceledTrace(t *testing.T) {
	store := newFakeStore()
	writer := NewWriter(store, fixedNow)

	result, err := writer.Cancel(context.Background(), &Request{TraceID: "trace_cancel", StartedAt: "2026-01-01T00:00:00Z"})
	if err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	if result.Trace.Status != model.TraceStatusCanceled || result.Trace.EndedAt == "" {
		t.Fatalf("trace = %#v, want canceled trace", result.Trace)
	}
}

func TestWriterRejectsInvalidInput(t *testing.T) {
	if _, err := (*Writer)(nil).Start(context.Background(), &Request{}); err == nil {
		t.Fatal("Start() error = nil, want nil writer error")
	}
	if _, err := NewWriter(nil, fixedNow).Start(context.Background(), &Request{}); err == nil {
		t.Fatal("Start() error = nil, want nil store error")
	}
	if _, err := NewWriter(newFakeStore(), fixedNow).Start(context.Background(), nil); err == nil {
		t.Fatal("Start() error = nil, want nil request error")
	}
	if _, err := NewWriter(newFakeStore(), fixedNow).Complete(context.Background(), &Request{}); err == nil {
		t.Fatal("Complete() error = nil, want missing trace id error")
	}
}

func TestWriterReturnsStoreFailure(t *testing.T) {
	store := newFakeStore()
	store.saveErr = errors.New("save failed")
	writer := NewWriter(store, fixedNow)

	_, err := writer.Start(context.Background(), &Request{})
	if err == nil {
		t.Fatal("Start() error = nil, want store error")
	}
	var traceErr *Error
	if !errors.As(err, &traceErr) || traceErr.Code != CodeTraceStoreFailed {
		t.Fatalf("Start() error = %#v, want CodeTraceStoreFailed", err)
	}
}

func TestWriterHonorsCanceledContext(t *testing.T) {
	store := newFakeStore()
	writer := NewWriter(store, fixedNow)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := writer.Start(ctx, &Request{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Start() error = %#v, want context.Canceled", err)
	}
	if len(store.traces) != 0 {
		t.Fatalf("stored traces = %#v, want none", store.traces)
	}
}

func requestWithResults() *Request {
	return &Request{
		TraceID:     "trace_done",
		StartedAt:   "2026-01-01T00:00:00Z",
		Hint:        "convert pdf",
		ProviderIDs: []string{"provider_cli"},
		Observations: []model.Observation{{
			ProviderID: "provider_cli",
			Type:       "usage",
		}},
		Proposal: &model.ProposalTrace{Model: "gpt-test"},
		Candidates: []model.Candidate{{
			ProviderID:   "provider_cli",
			CapabilityID: "document.convert",
			Description:  "Convert a document.",
			Execution: model.Execution{
				Kind: model.ExecutionKindCLI,
			},
		}},
		Probes: []model.Probe{{
			CandidateIndex: 0,
			Passed:         true,
			Verify: model.VerifySpec{
				Level:  model.VerifyLevelL1,
				Method: model.VerifyMethodExecute,
			},
			Evidence: []model.EvidenceRef{{ID: "evidence_file_exists"}},
		}},
		Promotions: []model.Promotion{{
			CandidateIndex: 0,
			CapabilityID:   "document.convert",
			BindingID:      "binding_abc",
			ProviderID:     "provider_cli",
		}},
	}
}

func fixedNow() time.Time {
	return time.Unix(123, 456).UTC()
}

type fakeStore struct {
	traces  []model.Trace
	saveErr error
}

func newFakeStore() *fakeStore {
	return &fakeStore{}
}

func (store *fakeStore) SaveTrace(trace *model.Trace) error {
	if store.saveErr != nil {
		return store.saveErr
	}
	store.traces = append(store.traces, *trace)
	return nil
}
