package cald

import (
	"path/filepath"
	"testing"
)

func TestEndpointFilePath(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	want := filepath.Join(home, connectionDirName, endpointFileName)
	if got := EndpointFilePath(home); got != want {
		t.Fatalf("EndpointFilePath() = %q, want %q", got, want)
	}
}
