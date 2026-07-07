package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestSaveGetAndListTraces(t *testing.T) {
	store := newTestStore(t)
	first := model.Trace{ID: "trace_b", Status: model.TraceStatusCompleted}
	second := model.Trace{ID: "trace_a", Status: model.TraceStatusRunning}

	if err := store.SaveTrace(&first); err != nil {
		t.Fatalf("SaveTrace(first) error = %v", err)
	}
	if err := store.SaveTrace(&second); err != nil {
		t.Fatalf("SaveTrace(second) error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(store.Root(), tracesDir, "trace_a", traceFileName)); err != nil {
		t.Fatalf("stat trace file: %v", err)
	}

	got, ok, err := store.GetTrace("trace_a")
	if err != nil {
		t.Fatalf("GetTrace() error = %v", err)
	}
	if !ok || got.ID != "trace_a" {
		t.Fatalf("GetTrace() = %#v, %v; want trace_a, true", got, ok)
	}

	traces, err := store.ListTraces()
	if err != nil {
		t.Fatalf("ListTraces() error = %v", err)
	}
	if len(traces) != 2 || traces[0].ID != "trace_a" || traces[1].ID != "trace_b" {
		t.Fatalf("ListTraces() = %#v, want sorted traces", traces)
	}
}

func TestTraceMissingEmptyAndIgnoredEntries(t *testing.T) {
	store := newTestStore(t)
	writeTestFile(t, filepath.Join(store.Root(), tracesDir, "note.txt"), "ignore me")
	if err := os.MkdirAll(filepath.Join(store.Root(), tracesDir, "trace_empty"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	traces, err := store.ListTraces()
	if err != nil {
		t.Fatalf("ListTraces() error = %v", err)
	}
	if len(traces) != 0 {
		t.Fatalf("ListTraces() len = %d, want 0", len(traces))
	}

	_, ok, err := store.GetTrace("trace_missing")
	if err != nil {
		t.Fatalf("GetTrace() error = %v", err)
	}
	if ok {
		t.Fatal("GetTrace() ok = true, want false")
	}
}

func TestTraceRejectsInvalidInputs(t *testing.T) {
	store := newTestStore(t)

	if err := store.SaveTrace(nil); err == nil {
		t.Fatal("SaveTrace(nil) error = nil, want error")
	}
	if err := store.SaveTrace(&model.Trace{ID: "../bad", Status: model.TraceStatusCompleted}); err == nil {
		t.Fatal("SaveTrace() error = nil, want path-safe id error")
	}
	if _, _, err := store.GetTrace("../bad"); err == nil {
		t.Fatal("GetTrace() error = nil, want path-safe id error")
	}
	if err := store.SaveTrace(&model.Trace{ID: "trace_bad", Status: "done"}); err == nil {
		t.Fatal("SaveTrace() error = nil, want validation error")
	}
}

func TestListTracesRejectsInvalidFiles(t *testing.T) {
	store := newTestStore(t)
	writeTestFile(t, filepath.Join(store.Root(), tracesDir, "trace_bad", traceFileName), "{")
	if _, err := store.ListTraces(); err == nil {
		t.Fatal("ListTraces() error = nil, want decode error")
	}

	store = newTestStore(t)
	writeTestFile(t, filepath.Join(store.Root(), tracesDir, "trace_bad", traceFileName), `{"id":"trace_bad","status":"done"}`)
	if _, err := store.ListTraces(); err == nil {
		t.Fatal("ListTraces() error = nil, want validation error")
	}
}
