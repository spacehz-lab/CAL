package store

import (
	"path/filepath"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestSaveGetAndListProviders(t *testing.T) {
	store := newTestStore(t)
	first := model.Provider{ID: "provider_b", Kind: model.ProviderKindCLI, Path: "/bin/b"}
	second := model.Provider{ID: "provider_a", Kind: model.ProviderKindCLI, Path: "/bin/a"}

	if err := store.SaveProvider(&first); err != nil {
		t.Fatalf("SaveProvider(first) error = %v", err)
	}
	if err := store.SaveProvider(&second); err != nil {
		t.Fatalf("SaveProvider(second) error = %v", err)
	}

	got, ok, err := store.GetProvider("provider_a")
	if err != nil {
		t.Fatalf("GetProvider() error = %v", err)
	}
	if !ok || got.ID != "provider_a" {
		t.Fatalf("GetProvider() = %#v, %v; want provider_a, true", got, ok)
	}

	providers, err := store.ListProviders()
	if err != nil {
		t.Fatalf("ListProviders() error = %v", err)
	}
	if len(providers) != 2 || providers[0].ID != "provider_a" || providers[1].ID != "provider_b" {
		t.Fatalf("ListProviders() = %#v, want sorted providers", providers)
	}
}

func TestProviderMissingAndEmptyList(t *testing.T) {
	store := newTestStore(t)

	providers, err := store.ListProviders()
	if err != nil {
		t.Fatalf("ListProviders() error = %v", err)
	}
	if len(providers) != 0 {
		t.Fatalf("ListProviders() len = %d, want 0", len(providers))
	}

	_, ok, err := store.GetProvider("provider_missing")
	if err != nil {
		t.Fatalf("GetProvider() error = %v", err)
	}
	if ok {
		t.Fatal("GetProvider() ok = true, want false")
	}
}

func TestProviderRejectsInvalidInputs(t *testing.T) {
	store := newTestStore(t)

	if err := store.SaveProvider(nil); err == nil {
		t.Fatal("SaveProvider(nil) error = nil, want error")
	}
	if err := store.SaveProvider(&model.Provider{ID: "../bad", Kind: model.ProviderKindCLI, Path: "/bin/bad"}); err == nil {
		t.Fatal("SaveProvider() error = nil, want path-safe id error")
	}
	if _, _, err := store.GetProvider("../bad"); err == nil {
		t.Fatal("GetProvider() error = nil, want path-safe id error")
	}
	if err := store.SaveProvider(&model.Provider{ID: "provider_bad", Kind: "bad", Path: "/bin/bad"}); err == nil {
		t.Fatal("SaveProvider() error = nil, want validation error")
	}
}

func TestListProvidersRejectsInvalidFiles(t *testing.T) {
	store := newTestStore(t)
	writeTestFile(t, filepath.Join(store.Root(), providersDir, "bad.json"), "{")
	if _, err := store.ListProviders(); err == nil {
		t.Fatal("ListProviders() error = nil, want decode error")
	}

	store = newTestStore(t)
	writeTestFile(t, filepath.Join(store.Root(), providersDir, "bad.json"), `{"id":"provider_bad","kind":"bad","path":"/bin/bad"}`)
	if _, err := store.ListProviders(); err == nil {
		t.Fatal("ListProviders() error = nil, want validation error")
	}
}
