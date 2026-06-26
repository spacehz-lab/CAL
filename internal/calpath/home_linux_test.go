package calpath

import (
	"path/filepath"
	"testing"
)

func TestHomeDirUsesXDGDataHome(t *testing.T) {
	t.Setenv(envHome, "")
	t.Setenv("XDG_DATA_HOME", "/data")

	got, err := HomeDir()
	if err != nil {
		t.Fatalf("HomeDir() error = %v", err)
	}
	want := filepath.Join("/data", "cal")
	if got != want {
		t.Fatalf("HomeDir() = %q, want %q", got, want)
	}
}

func TestHomeDirFallsBackToLocalShare(t *testing.T) {
	t.Setenv(envHome, "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "/home/test")

	got, err := HomeDir()
	if err != nil {
		t.Fatalf("HomeDir() error = %v", err)
	}
	want := filepath.Join("/home/test", ".local", "share", "cal")
	if got != want {
		t.Fatalf("HomeDir() = %q, want %q", got, want)
	}
}
