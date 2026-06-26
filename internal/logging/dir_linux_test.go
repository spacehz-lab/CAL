package logging

import (
	"path/filepath"
	"testing"
)

func TestDefaultLogDirUsesXDGStateDirectory(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/state")

	dir, err := defaultLogDir()
	if err != nil {
		t.Fatalf("defaultLogDir() error = %v", err)
	}
	want := filepath.Join("/state", "cal", "logs")
	if dir != want {
		t.Fatalf("defaultLogDir() = %q, want %q", dir, want)
	}
}

func TestDefaultLogDirFallsBackToLocalState(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", "/home/test")

	dir, err := defaultLogDir()
	if err != nil {
		t.Fatalf("defaultLogDir() error = %v", err)
	}
	want := filepath.Join("/home/test", ".local", "state", "cal", "logs")
	if dir != want {
		t.Fatalf("defaultLogDir() = %q, want %q", dir, want)
	}
}
