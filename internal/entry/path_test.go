package entry

import (
	"path/filepath"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestNormalizeProviderPathExpandsEnvironment(t *testing.T) {
	path := writeExecutable(t, "provider")
	t.Setenv("CAL_ENTRY_PROVIDER", path)

	normalized, err := normalizeProviderPath("$CAL_ENTRY_PROVIDER")
	if err != nil {
		t.Fatalf("normalizeProviderPath() error = %v", err)
	}
	want, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	if normalized != want {
		t.Fatalf("normalized = %q, want %q", normalized, want)
	}
}

func TestGetByPathNormalizesWithoutStat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing")
	normalized, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	provider := testProvider(normalized)
	store := newFakeStore()
	store.providers[provider.ID] = provider

	got, ok, err := NewRegistry(store).GetByPath(nil, path)
	if err != nil {
		t.Fatalf("GetByPath() error = %v", err)
	}
	if !ok || got.Path != normalized {
		t.Fatalf("GetByPath() = %#v, %v, want normalized stored provider", got, ok)
	}
}

func TestProviderNameForApp(t *testing.T) {
	if got := providerName("/Applications/Preview.app", model.ProviderKindApp); got != "Preview" {
		t.Fatalf("providerName() = %q, want Preview", got)
	}
}

func testProvider(path string) model.Provider {
	return model.Provider{
		ID:   "provider_test",
		Name: filepath.Base(path),
		Kind: model.ProviderKindCLI,
		Path: path,
	}
}
