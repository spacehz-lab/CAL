package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewRequiresRoot(t *testing.T) {
	if _, err := New(" "); err == nil {
		t.Fatal("New() error = nil, want missing root error")
	}
}

func TestEnsureCreatesStoreDirectories(t *testing.T) {
	store := newTestStore(t)
	if err := store.Ensure(); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}

	for _, dir := range []string{providersDir, capabilitiesDir, tracesDir, runsDir} {
		info, err := os.Stat(filepath.Join(store.Root(), dir))
		if err != nil {
			t.Fatalf("stat %s: %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("%s is not a directory", dir)
		}
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return store
}

func writeTestFile(t *testing.T, path string, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
