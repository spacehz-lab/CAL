package store

import (
	"path/filepath"
	"testing"
)

func TestOpenFromEnvUsesCALHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CAL_HOME", home)

	store, err := OpenFromEnv()
	if err != nil {
		t.Fatalf("OpenFromEnv() error = %v", err)
	}
	if store.Home() != filepath.Clean(home) {
		t.Fatalf("Home() = %q, want %q", store.Home(), filepath.Clean(home))
	}
}
