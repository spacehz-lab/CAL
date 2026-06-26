package calpath

import (
	"path/filepath"
	"testing"
)

func TestHomeDirUsesDarwinApplicationSupport(t *testing.T) {
	t.Setenv(envHome, "")
	t.Setenv("HOME", "/Users/test")

	got, err := HomeDir()
	if err != nil {
		t.Fatalf("HomeDir() error = %v", err)
	}
	want := filepath.Join("/Users/test", "Library", "Application Support", "cal")
	if got != want {
		t.Fatalf("HomeDir() = %q, want %q", got, want)
	}
}
