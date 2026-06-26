package store

import (
	"path/filepath"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

func TestPutAndListProvider(t *testing.T) {
	store := newTestStore(t)
	first := core.Provider{
		ID:   "provider_b",
		Name: "textutil",
		Kind: core.ProviderKindCLI,
		Path: "/usr/bin/textutil",
	}
	second := core.Provider{
		ID:   "provider_a",
		Name: "sips",
		Kind: core.ProviderKindCLI,
		Path: "/usr/bin/sips",
	}

	if err := store.PutProvider(first); err != nil {
		t.Fatalf("PutProvider(first) error = %v", err)
	}
	if err := store.PutProvider(second); err != nil {
		t.Fatalf("PutProvider(second) error = %v", err)
	}
	writeStoreFile(t, filepath.Join(store.Home(), providersDir, "ignore.txt"), "{}")

	providers, err := store.ListProviders()
	if err != nil {
		t.Fatalf("ListProviders() error = %v", err)
	}
	if len(providers) != 2 || providers[0].ID != "provider_a" || providers[1].ID != "provider_b" {
		t.Fatalf("ListProviders() = %#v, want sorted providers", providers)
	}
}

func TestGetProvider(t *testing.T) {
	store := newTestStore(t)
	provider := core.Provider{ID: "provider_one", Kind: core.ProviderKindCLI, Path: "/tmp/one"}
	if err := store.PutProvider(provider); err != nil {
		t.Fatalf("PutProvider() error = %v", err)
	}

	got, ok, err := store.GetProvider("provider_one")
	if err != nil {
		t.Fatalf("GetProvider() error = %v", err)
	}
	if !ok || got.ID != provider.ID {
		t.Fatalf("GetProvider() = %#v, %v, want provider", got, ok)
	}
}

func TestListProvidersEmptyStore(t *testing.T) {
	providers, err := newTestStore(t).ListProviders()
	if err != nil {
		t.Fatalf("ListProviders() error = %v", err)
	}
	if len(providers) != 0 {
		t.Fatalf("ListProviders() len = %d, want 0", len(providers))
	}
}

func TestPutProviderRejectsInvalidRecord(t *testing.T) {
	if err := newTestStore(t).PutProvider(core.Provider{ID: "provider_bad"}); err == nil {
		t.Fatal("PutProvider(invalid) error = nil, want error")
	}
}

func TestListProvidersRejectsInvalidJSON(t *testing.T) {
	store := newTestStore(t)
	writeStoreFile(t, filepath.Join(store.Home(), providersDir, "bad.json"), "{")

	if _, err := store.ListProviders(); err == nil {
		t.Fatal("ListProviders() error = nil, want decode error")
	}
}

func TestListProvidersRejectsInvalidRecord(t *testing.T) {
	store := newTestStore(t)
	writeStoreFile(t, filepath.Join(store.Home(), providersDir, "bad.json"), `{"id":"provider_bad","kind":"cli"}`)

	if _, err := store.ListProviders(); err == nil {
		t.Fatal("ListProviders() error = nil, want validation error")
	}
}
