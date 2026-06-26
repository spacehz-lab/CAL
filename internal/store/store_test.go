package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenRejectsBlankHome(t *testing.T) {
	if _, err := Open(" "); err == nil {
		t.Fatal("Open(blank) error = nil, want error")
	}
}

func TestEnsureCreatesStoreDirectories(t *testing.T) {
	store := newTestStore(t)
	if err := store.Ensure(); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}

	for _, dir := range []string{providersDir, capabilitiesDir, discoveryDir, runsDir} {
		if info, err := os.Stat(filepath.Join(store.Home(), dir)); err != nil {
			t.Fatalf("%s dir missing: %v", dir, err)
		} else if !info.IsDir() {
			t.Fatalf("%s is not a directory", dir)
		}
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()

	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	return store
}

func writeStoreFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}
