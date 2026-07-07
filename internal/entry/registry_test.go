package entry

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestRegisterCreatesAndUpdatesProvider(t *testing.T) {
	store := newFakeStore()
	path := writeExecutable(t, "provider")
	registry := NewRegistry(store)

	first, err := registry.Register(context.Background(), &RegisterRequest{ProviderPath: path})
	if err != nil {
		t.Fatalf("Register(first) error = %v", err)
	}
	if !first.Created || first.Updated {
		t.Fatalf("first result = %#v, want created", first)
	}
	if first.Provider.Kind != model.ProviderKindCLI || first.Provider.Path == "" || !strings.HasPrefix(first.Provider.ID, "provider_") {
		t.Fatalf("provider = %#v, want CLI provider with stable id", first.Provider)
	}
	if first.Provider.Name != filepath.Base(path) {
		t.Fatalf("provider name = %q, want basename", first.Provider.Name)
	}

	second, err := registry.Register(context.Background(), &RegisterRequest{ProviderPath: path})
	if err != nil {
		t.Fatalf("Register(second) error = %v", err)
	}
	if second.Created || !second.Updated {
		t.Fatalf("second result = %#v, want updated", second)
	}
	if len(store.saved) != 2 {
		t.Fatalf("saved count = %d, want 2", len(store.saved))
	}
}

func TestRegisterRejectsInvalidAndUnsupportedPath(t *testing.T) {
	registry := NewRegistry(newFakeStore())

	_, err := registry.Register(context.Background(), &RegisterRequest{})
	assertEntryCode(t, err, CodeInvalidProviderPath)

	_, err = registry.Register(context.Background(), &RegisterRequest{ProviderPath: t.TempDir()})
	assertEntryCode(t, err, CodeTargetProviderNotFound)

	missing := filepath.Join(t.TempDir(), "missing")
	_, err = registry.Register(context.Background(), &RegisterRequest{ProviderPath: missing})
	assertEntryCode(t, err, CodeTargetProviderNotFound)
}

func TestLoadListAndGetByPath(t *testing.T) {
	path := writeExecutable(t, "provider")
	normalized, err := normalizeProviderPath(path)
	if err != nil {
		t.Fatalf("normalizeProviderPath() error = %v", err)
	}
	provider := model.Provider{
		ID:   model.ProviderID(runtime.GOOS, model.ProviderKindCLI, normalized),
		Name: "provider",
		Kind: model.ProviderKindCLI,
		Path: normalized,
	}
	store := newFakeStore()
	store.providers[provider.ID] = provider
	registry := NewRegistry(store)

	loaded, err := registry.Load(context.Background(), &LoadRequest{ProviderID: " " + provider.ID + " "})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Provider.ID != provider.ID {
		t.Fatalf("loaded provider = %#v, want %s", loaded.Provider, provider.ID)
	}

	providers, err := registry.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(providers) != 1 || providers[0].ID != provider.ID {
		t.Fatalf("providers = %#v, want provider", providers)
	}

	if err := os.Remove(path); err != nil {
		t.Fatalf("remove provider file: %v", err)
	}
	got, ok, err := registry.GetByPath(context.Background(), path)
	if err != nil {
		t.Fatalf("GetByPath() error = %v", err)
	}
	if !ok || got.ID != provider.ID {
		t.Fatalf("GetByPath() = %#v, %v, want stored provider", got, ok)
	}
}

func TestLoadMissingProvider(t *testing.T) {
	registry := NewRegistry(newFakeStore())

	_, err := registry.Load(context.Background(), &LoadRequest{ProviderID: "provider_missing"})
	assertEntryCode(t, err, CodeProviderNotFound)

	_, err = registry.Load(context.Background(), &LoadRequest{})
	assertEntryCode(t, err, CodeProviderNotFound)
}

func TestStoreErrors(t *testing.T) {
	t.Run("register list", func(t *testing.T) {
		store := newFakeStore()
		store.listErr = errors.New("list failed")
		_, err := NewRegistry(store).Register(context.Background(), &RegisterRequest{ProviderPath: writeExecutable(t, "provider")})
		assertEntryCode(t, err, CodeEntryStoreFailed)
	})

	t.Run("register save", func(t *testing.T) {
		store := newFakeStore()
		store.saveErr = errors.New("save failed")
		_, err := NewRegistry(store).Register(context.Background(), &RegisterRequest{ProviderPath: writeExecutable(t, "provider")})
		assertEntryCode(t, err, CodeEntryStoreFailed)
	})

	t.Run("load get", func(t *testing.T) {
		store := newFakeStore()
		store.getErr = errors.New("get failed")
		_, err := NewRegistry(store).Load(context.Background(), &LoadRequest{ProviderID: "provider_test"})
		assertEntryCode(t, err, CodeEntryStoreFailed)
	})

	t.Run("list", func(t *testing.T) {
		store := newFakeStore()
		store.listErr = errors.New("list failed")
		_, err := NewRegistry(store).List(context.Background())
		assertEntryCode(t, err, CodeEntryStoreFailed)
	})

	t.Run("get by path", func(t *testing.T) {
		store := newFakeStore()
		store.listErr = errors.New("list failed")
		_, _, err := NewRegistry(store).GetByPath(context.Background(), writeExecutable(t, "provider"))
		assertEntryCode(t, err, CodeEntryStoreFailed)
	})
}

func TestContextCancellationStopsBeforeWork(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	store := newFakeStore()

	_, err := NewRegistry(store).Register(ctx, &RegisterRequest{ProviderPath: writeExecutable(t, "provider")})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Register() error = %v, want context canceled", err)
	}
	if store.listCalls != 0 || store.saveCalls != 0 {
		t.Fatalf("store calls = list %d save %d, want none", store.listCalls, store.saveCalls)
	}
}

func TestRegistryRequiresStore(t *testing.T) {
	_, err := NewRegistry(nil).List(context.Background())
	assertEntryCode(t, err, CodeEntryStoreFailed)
}

func assertEntryCode(t *testing.T, err error, code string) {
	t.Helper()
	var entryErr *Error
	if !errors.As(err, &entryErr) {
		t.Fatalf("error = %v, want entry.Error code %s", err, code)
	}
	if entryErr.Code != code {
		t.Fatalf("code = %s, want %s", entryErr.Code, code)
	}
}

type fakeStore struct {
	providers map[string]model.Provider
	saved     []model.Provider

	listErr error
	getErr  error
	saveErr error

	listCalls int
	saveCalls int
}

func newFakeStore() *fakeStore {
	return &fakeStore{providers: map[string]model.Provider{}}
}

func (store *fakeStore) ListProviders() ([]model.Provider, error) {
	store.listCalls++
	if store.listErr != nil {
		return nil, store.listErr
	}
	providers := make([]model.Provider, 0, len(store.providers))
	for _, provider := range store.providers {
		providers = append(providers, provider)
	}
	return providers, nil
}

func (store *fakeStore) GetProvider(id string) (model.Provider, bool, error) {
	if store.getErr != nil {
		return model.Provider{}, false, store.getErr
	}
	provider, ok := store.providers[id]
	return provider, ok, nil
}

func (store *fakeStore) SaveProvider(provider *model.Provider) error {
	store.saveCalls++
	if store.saveErr != nil {
		return store.saveErr
	}
	store.providers[provider.ID] = *provider
	store.saved = append(store.saved, *provider)
	return nil
}

func writeExecutable(t *testing.T, name string) string {
	t.Helper()
	if runtime.GOOS == "windows" && filepath.Ext(name) == "" {
		name += ".exe"
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("test"), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}
	return path
}
