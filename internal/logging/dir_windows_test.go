package logging

import (
	"path/filepath"
	"testing"
)

func TestDefaultDirWindowsUsesLocalAppData(t *testing.T) {
	t.Setenv(envLocalAppData, `C:\Users\test\AppData\Local`)

	got, err := defaultDir()
	if err != nil {
		t.Fatalf("defaultDir() error = %v", err)
	}
	want := filepath.Join(`C:\Users\test\AppData\Local`, defaultName, "logs")
	if got != want {
		t.Fatalf("defaultDir() = %q, want %q", got, want)
	}
}
