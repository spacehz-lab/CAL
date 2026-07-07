package store

import (
	"path/filepath"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestSaveGetAndListCapabilities(t *testing.T) {
	store := newTestStore(t)
	first := testCapability("document.merge")
	second := testCapability("document.convert")

	if err := store.SaveCapability(&first); err != nil {
		t.Fatalf("SaveCapability(first) error = %v", err)
	}
	if err := store.SaveCapability(&second); err != nil {
		t.Fatalf("SaveCapability(second) error = %v", err)
	}

	got, ok, err := store.GetCapability("document.convert")
	if err != nil {
		t.Fatalf("GetCapability() error = %v", err)
	}
	if !ok || got.ID != "document.convert" {
		t.Fatalf("GetCapability() = %#v, %v; want document.convert, true", got, ok)
	}

	capabilities, err := store.ListCapabilities()
	if err != nil {
		t.Fatalf("ListCapabilities() error = %v", err)
	}
	if len(capabilities) != 2 || capabilities[0].ID != "document.convert" || capabilities[1].ID != "document.merge" {
		t.Fatalf("ListCapabilities() = %#v, want sorted capabilities", capabilities)
	}
}

func TestCapabilityMissingAndEmptyList(t *testing.T) {
	store := newTestStore(t)

	capabilities, err := store.ListCapabilities()
	if err != nil {
		t.Fatalf("ListCapabilities() error = %v", err)
	}
	if len(capabilities) != 0 {
		t.Fatalf("ListCapabilities() len = %d, want 0", len(capabilities))
	}

	_, ok, err := store.GetCapability("document.missing")
	if err != nil {
		t.Fatalf("GetCapability() error = %v", err)
	}
	if ok {
		t.Fatal("GetCapability() ok = true, want false")
	}
}

func TestCapabilityRejectsInvalidInputs(t *testing.T) {
	store := newTestStore(t)

	if err := store.SaveCapability(nil); err == nil {
		t.Fatal("SaveCapability(nil) error = nil, want error")
	}
	if err := store.SaveCapability(&model.Capability{ID: "../bad"}); err == nil {
		t.Fatal("SaveCapability() error = nil, want validation or path-safe id error")
	}
	if _, _, err := store.GetCapability("../bad"); err == nil {
		t.Fatal("GetCapability() error = nil, want path-safe id error")
	}
	if err := store.SaveCapability(&model.Capability{ID: "bad"}); err == nil {
		t.Fatal("SaveCapability() error = nil, want validation error")
	}
}

func TestListCapabilitiesRejectsInvalidFiles(t *testing.T) {
	store := newTestStore(t)
	writeTestFile(t, filepath.Join(store.Root(), capabilitiesDir, "bad.json"), "{")
	if _, err := store.ListCapabilities(); err == nil {
		t.Fatal("ListCapabilities() error = nil, want decode error")
	}

	store = newTestStore(t)
	writeTestFile(t, filepath.Join(store.Root(), capabilitiesDir, "bad.json"), `{"id":"bad"}`)
	if _, err := store.ListCapabilities(); err == nil {
		t.Fatal("ListCapabilities() error = nil, want validation error")
	}
}

func testCapability(id string) model.Capability {
	return model.Capability{
		ID:          id,
		Description: "Export a document.",
		Bindings: []model.Binding{
			{
				ID:           "binding_" + id,
				CapabilityID: id,
				ProviderID:   "provider_cli",
				Execution:    model.Execution{Kind: model.ExecutionKindCLI},
				Verify:       &model.VerifySpec{Level: model.VerifyLevelL1, Method: model.VerifyMethodExecute},
				Evidence:     []model.EvidenceRef{{ID: "evidence_" + id}},
				State:        model.BindingStatePromoted,
			},
		},
	}
}
