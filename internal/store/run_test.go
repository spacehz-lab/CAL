package store

import (
	"path/filepath"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestSaveGetAndListRuns(t *testing.T) {
	store := newTestStore(t)
	first := model.Run{ID: "run_b", CapabilityID: "document.convert", Status: model.RunStatusSucceeded}
	second := model.Run{ID: "run_a", CapabilityID: "document.convert", Status: model.RunStatusFailed}

	if err := store.SaveRun(&first); err != nil {
		t.Fatalf("SaveRun(first) error = %v", err)
	}
	if err := store.SaveRun(&second); err != nil {
		t.Fatalf("SaveRun(second) error = %v", err)
	}

	got, ok, err := store.GetRun("run_a")
	if err != nil {
		t.Fatalf("GetRun() error = %v", err)
	}
	if !ok || got.ID != "run_a" {
		t.Fatalf("GetRun() = %#v, %v; want run_a, true", got, ok)
	}

	runs, err := store.ListRuns()
	if err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}
	if len(runs) != 2 || runs[0].ID != "run_a" || runs[1].ID != "run_b" {
		t.Fatalf("ListRuns() = %#v, want sorted runs", runs)
	}
}

func TestRunMissingAndEmptyList(t *testing.T) {
	store := newTestStore(t)

	runs, err := store.ListRuns()
	if err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}
	if len(runs) != 0 {
		t.Fatalf("ListRuns() len = %d, want 0", len(runs))
	}

	_, ok, err := store.GetRun("run_missing")
	if err != nil {
		t.Fatalf("GetRun() error = %v", err)
	}
	if ok {
		t.Fatal("GetRun() ok = true, want false")
	}
}

func TestRunRejectsInvalidInputs(t *testing.T) {
	store := newTestStore(t)

	if err := store.SaveRun(nil); err == nil {
		t.Fatal("SaveRun(nil) error = nil, want error")
	}
	if err := store.SaveRun(&model.Run{ID: "../bad", CapabilityID: "document.convert", Status: model.RunStatusSucceeded}); err == nil {
		t.Fatal("SaveRun() error = nil, want path-safe id error")
	}
	if _, _, err := store.GetRun("../bad"); err == nil {
		t.Fatal("GetRun() error = nil, want path-safe id error")
	}
	if err := store.SaveRun(&model.Run{ID: "run_bad", CapabilityID: "document.convert", Status: "done"}); err == nil {
		t.Fatal("SaveRun() error = nil, want validation error")
	}
}

func TestListRunsRejectsInvalidFiles(t *testing.T) {
	store := newTestStore(t)
	writeTestFile(t, filepath.Join(store.Root(), runsDir, "bad.json"), "{")
	if _, err := store.ListRuns(); err == nil {
		t.Fatal("ListRuns() error = nil, want decode error")
	}

	store = newTestStore(t)
	writeTestFile(t, filepath.Join(store.Root(), runsDir, "bad.json"), `{"id":"run_bad","capability_id":"document.convert","status":"done"}`)
	if _, err := store.ListRuns(); err == nil {
		t.Fatal("ListRuns() error = nil, want validation error")
	}
}
