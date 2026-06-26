package logging

import (
	"path/filepath"
	"testing"
)

func TestDefaultLogDirUsesLocalAppData(t *testing.T) {
	t.Setenv("LocalAppData", `C:\Users\test\AppData\Local`)

	dir, err := defaultLogDir()
	if err != nil {
		t.Fatalf("defaultLogDir() error = %v", err)
	}
	want := filepath.Join(`C:\Users\test\AppData\Local`, "cal", "logs")
	if dir != want {
		t.Fatalf("defaultLogDir() = %q, want %q", dir, want)
	}
}
