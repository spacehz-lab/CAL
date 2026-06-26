package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteJSONAtomicRejectsUnsupportedValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := writeJSONAtomic(path, func() {}); err == nil {
		t.Fatal("writeJSONAtomic(func) error = nil, want error")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("unsupported value wrote target file or stat failed unexpectedly: %v", err)
	}
}
