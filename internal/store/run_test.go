package store

import (
	"path/filepath"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

func TestPutAndListRun(t *testing.T) {
	store := newTestStore(t)
	if err := store.PutRun(core.Run{
		ID:           "run_b",
		CapabilityID: "document.export_pdf",
		Status:       core.RunStatusFailed,
	}); err != nil {
		t.Fatalf("PutRun(run_b) error = %v", err)
	}
	if err := store.PutRun(core.Run{
		ID:           "run_a",
		CapabilityID: "document.export_pdf",
		Status:       core.RunStatusSucceeded,
		Verified:     true,
	}); err != nil {
		t.Fatalf("PutRun(run_a) error = %v", err)
	}

	runs, err := store.ListRuns()
	if err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}
	if len(runs) != 2 || runs[0].ID != "run_a" || runs[1].ID != "run_b" {
		t.Fatalf("ListRuns() = %#v, want sorted runs", runs)
	}
}

func TestGetRun(t *testing.T) {
	store := newTestStore(t)
	run := core.Run{
		ID:           "run_one",
		CapabilityID: "document.export_pdf",
		Status:       core.RunStatusSucceeded,
	}
	if err := store.PutRun(run); err != nil {
		t.Fatalf("PutRun() error = %v", err)
	}

	got, ok, err := store.GetRun("run_one")
	if err != nil {
		t.Fatalf("GetRun() error = %v", err)
	}
	if !ok || got.ID != run.ID {
		t.Fatalf("GetRun() = %#v, %v, want run", got, ok)
	}
}

func TestListRunsEmptyStore(t *testing.T) {
	runs, err := newTestStore(t).ListRuns()
	if err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}
	if len(runs) != 0 {
		t.Fatalf("ListRuns() len = %d, want 0", len(runs))
	}
}

func TestPutRunRejectsInvalidRecord(t *testing.T) {
	if err := newTestStore(t).PutRun(core.Run{ID: "run_bad"}); err == nil {
		t.Fatal("PutRun(invalid) error = nil, want error")
	}
}

func TestListRunsRejectsInvalidJSON(t *testing.T) {
	store := newTestStore(t)
	writeStoreFile(t, filepath.Join(store.Home(), runsDir, "bad.json"), "{")

	if _, err := store.ListRuns(); err == nil {
		t.Fatal("ListRuns() error = nil, want decode error")
	}
}

func TestListRunsRejectsInvalidRecord(t *testing.T) {
	store := newTestStore(t)
	writeStoreFile(t, filepath.Join(store.Home(), runsDir, "bad.json"), `{"id":"run_bad","capability_id":"document.export_pdf"}`)

	if _, err := store.ListRuns(); err == nil {
		t.Fatal("ListRuns() error = nil, want validation error")
	}
}
