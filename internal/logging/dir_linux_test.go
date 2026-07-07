package logging

import (
	"path/filepath"
	"testing"
)

func TestDefaultDirLinuxUsesXDGStateHome(t *testing.T) {
	t.Setenv(envXDGStateHome, "/state")

	got, err := defaultDir()
	if err != nil {
		t.Fatalf("defaultDir() error = %v", err)
	}
	want := filepath.Join("/state", defaultName, "logs")
	if got != want {
		t.Fatalf("defaultDir() = %q, want %q", got, want)
	}
}

func TestDefaultDirLinuxFallsBackToUserState(t *testing.T) {
	t.Setenv(envXDGStateHome, "")
	t.Setenv("HOME", "/home/test")

	got, err := defaultDir()
	if err != nil {
		t.Fatalf("defaultDir() error = %v", err)
	}
	want := filepath.Join("/home/test", ".local", "state", defaultName, "logs")
	if got != want {
		t.Fatalf("defaultDir() = %q, want %q", got, want)
	}
}
