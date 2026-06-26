package store

import (
	"os"
	"path/filepath"
	"testing"

	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestPutGetAndListTrace(t *testing.T) {
	store := newTestStore(t)
	trace := caltrace.Trace{
		ID:          "trace_abc123",
		Status:      caltrace.StatusCompleted,
		ProviderIDs: []string{"provider_abc123"},
	}
	if err := store.PutTrace(trace); err != nil {
		t.Fatalf("PutTrace() error = %v", err)
	}

	got, ok, err := store.GetTrace("trace_abc123")
	if err != nil {
		t.Fatalf("GetTrace() error = %v", err)
	}
	if !ok || got.ID != "trace_abc123" {
		t.Fatalf("GetTrace() = %#v, %v, want trace_abc123", got, ok)
	}

	traces, err := store.ListTraces()
	if err != nil {
		t.Fatalf("ListTraces() error = %v", err)
	}
	if len(traces) != 1 || traces[0].ID != "trace_abc123" {
		t.Fatalf("ListTraces() = %#v, want trace_abc123", traces)
	}
	if _, err := os.Stat(filepath.Join(store.Home(), "discovery", "trace_abc123", "trace.json")); err != nil {
		t.Fatalf("trace file missing: %v", err)
	}
}

func TestPrepareTraceProbeDir(t *testing.T) {
	store := newTestStore(t)
	dir, err := store.PrepareTraceProbeDir("trace_abc123", 2)
	if err != nil {
		t.Fatalf("PrepareTraceProbeDir() error = %v", err)
	}
	want := filepath.Join(store.Home(), "discovery", "trace_abc123", "probes", "2")
	if dir != want {
		t.Fatalf("PrepareTraceProbeDir() = %q, want %q", dir, want)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("probe dir stat error = %v", err)
	}
}

func TestPrepareTraceProbeDirRejectsInvalidInput(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.PrepareTraceProbeDir("../bad", 0); err == nil {
		t.Fatal("PrepareTraceProbeDir(unsafe) error = nil, want error")
	}
	if _, err := store.PrepareTraceProbeDir("trace_abc123", -1); err == nil {
		t.Fatal("PrepareTraceProbeDir(negative) error = nil, want error")
	}
}

func TestGetTraceMissing(t *testing.T) {
	trace, ok, err := newTestStore(t).GetTrace("trace_missing")
	if err != nil {
		t.Fatalf("GetTrace(missing) error = %v", err)
	}
	if ok || trace.ID != "" {
		t.Fatalf("GetTrace(missing) = %#v, %v, want not found", trace, ok)
	}
}

func TestPutTraceRejectsInvalidRecord(t *testing.T) {
	if err := newTestStore(t).PutTrace(caltrace.Trace{ID: "trace_bad"}); err == nil {
		t.Fatal("PutTrace(invalid) error = nil, want error")
	}
}

func TestPutTraceRejectsUnsafeID(t *testing.T) {
	if err := newTestStore(t).PutTrace(caltrace.Trace{ID: "../bad", Status: caltrace.StatusCompleted}); err == nil {
		t.Fatal("PutTrace(unsafe) error = nil, want error")
	}
}

func TestGetTraceRejectsUnsafeID(t *testing.T) {
	if _, _, err := newTestStore(t).GetTrace("../bad"); err == nil {
		t.Fatal("GetTrace(unsafe) error = nil, want error")
	}
}

func TestGetTraceRejectsInvalidStoredRecord(t *testing.T) {
	store := newTestStore(t)
	writeStoreFile(t, filepath.Join(store.Home(), discoveryDir, "trace_bad", "trace.json"), `{"id":"trace_bad"}`)

	if _, _, err := store.GetTrace("trace_bad"); err == nil {
		t.Fatal("GetTrace(invalid stored record) error = nil, want error")
	}
}

func TestListTracesEmptyStore(t *testing.T) {
	traces, err := newTestStore(t).ListTraces()
	if err != nil {
		t.Fatalf("ListTraces() error = %v", err)
	}
	if len(traces) != 0 {
		t.Fatalf("ListTraces() len = %d, want 0", len(traces))
	}
}

func TestListTracesSkipsFiles(t *testing.T) {
	store := newTestStore(t)
	writeStoreFile(t, filepath.Join(store.Home(), discoveryDir, "ignore.json"), "{}")

	traces, err := store.ListTraces()
	if err != nil {
		t.Fatalf("ListTraces() error = %v", err)
	}
	if len(traces) != 0 {
		t.Fatalf("ListTraces() = %#v, want empty", traces)
	}
}
