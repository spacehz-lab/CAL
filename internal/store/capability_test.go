package store

import (
	"path/filepath"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

func TestPutGetAndListCapability(t *testing.T) {
	store := newTestStore(t)
	capability := validCapability("document.export_pdf")
	if err := store.PutCapability(capability); err != nil {
		t.Fatalf("PutCapability() error = %v", err)
	}

	got, ok, err := store.GetCapability("document.export_pdf")
	if err != nil {
		t.Fatalf("GetCapability() error = %v", err)
	}
	if !ok || got.ID != "document.export_pdf" {
		t.Fatalf("GetCapability() = %#v, %v, want document.export_pdf", got, ok)
	}

	missing, ok, err := store.GetCapability("document.missing")
	if err != nil {
		t.Fatalf("GetCapability(missing) error = %v", err)
	}
	if ok || missing.ID != "" {
		t.Fatalf("GetCapability(missing) = %#v, %v, want not found", missing, ok)
	}

	capabilities, err := store.ListCapabilities()
	if err != nil {
		t.Fatalf("ListCapabilities() error = %v", err)
	}
	if len(capabilities) != 1 || capabilities[0].ID != "document.export_pdf" {
		t.Fatalf("ListCapabilities() = %#v, want document.export_pdf", capabilities)
	}
}

func TestListCapabilitiesEmptyStore(t *testing.T) {
	capabilities, err := newTestStore(t).ListCapabilities()
	if err != nil {
		t.Fatalf("ListCapabilities() error = %v", err)
	}
	if len(capabilities) != 0 {
		t.Fatalf("ListCapabilities() len = %d, want 0", len(capabilities))
	}
}

func TestPutCapabilityRejectsUnsafeID(t *testing.T) {
	if err := newTestStore(t).PutCapability(core.Capability{ID: "../bad"}); err == nil {
		t.Fatal("PutCapability() error = nil, want unsafe id error")
	}
}

func TestGetCapabilityRejectsUnsafeID(t *testing.T) {
	if _, _, err := newTestStore(t).GetCapability("../bad"); err == nil {
		t.Fatal("GetCapability(unsafe) error = nil, want error")
	}
}

func TestGetCapabilityRejectsInvalidStoredRecord(t *testing.T) {
	store := newTestStore(t)
	writeStoreFile(t, filepath.Join(store.Home(), capabilitiesDir, "document.bad.json"), `{"id":"document.bad","bindings":[{"id":"binding_bad"}]}`)

	if _, _, err := store.GetCapability("document.bad"); err == nil {
		t.Fatal("GetCapability(invalid stored record) error = nil, want error")
	}
}

func TestListCapabilitiesRejectsInvalidJSON(t *testing.T) {
	store := newTestStore(t)
	writeStoreFile(t, filepath.Join(store.Home(), capabilitiesDir, "bad.json"), "{")

	if _, err := store.ListCapabilities(); err == nil {
		t.Fatal("ListCapabilities() error = nil, want decode error")
	}
}

func TestListCapabilitiesRejectsInvalidRecord(t *testing.T) {
	store := newTestStore(t)
	writeStoreFile(t, filepath.Join(store.Home(), capabilitiesDir, "bad.json"), `{"id":"bad"}`)

	if _, err := store.ListCapabilities(); err == nil {
		t.Fatal("ListCapabilities() error = nil, want validation error")
	}
}

func validCapability(id string) core.Capability {
	return core.Capability{
		ID:          id,
		Description: "Export a document to a PDF artifact.",
		Bindings: []core.Binding{{
			ID:           "binding_abc123",
			CapabilityID: id,
			ProviderID:   "provider_abc123",
			Execution:    core.Execution{Kind: core.ExecutionKindCLI},
			Verifier:     &core.Verifier{ID: "file_exists"},
			Evidence:     []core.EvidenceRef{{ID: "evidence_abc123"}},
			State:        core.BindingStatePromoted,
		}},
	}
}
