package logging

import (
	"path/filepath"
	"testing"
)

func TestDefaultDirDarwin(t *testing.T) {
	t.Setenv("HOME", "/Users/test")

	got, err := defaultDir()
	if err != nil {
		t.Fatalf("defaultDir() error = %v", err)
	}
	want := filepath.Join("/Users/test", "Library", "Logs", defaultName)
	if got != want {
		t.Fatalf("defaultDir() = %q, want %q", got, want)
	}
}
