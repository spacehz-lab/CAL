package logging

import (
	"path/filepath"
	"testing"
)

func TestDefaultLogDirUsesDarwinLogsDirectory(t *testing.T) {
	t.Setenv("HOME", "/Users/test")

	dir, err := defaultLogDir()
	if err != nil {
		t.Fatalf("defaultLogDir() error = %v", err)
	}
	want := filepath.Join("/Users/test", "Library", "Logs", "cal")
	if dir != want {
		t.Fatalf("defaultLogDir() = %q, want %q", dir, want)
	}
}
