package calpath

import (
	"path/filepath"
	"testing"
)

func TestHomeDirUsesLocalAppData(t *testing.T) {
	t.Setenv(envHome, "")
	t.Setenv("LocalAppData", `C:\Users\test\AppData\Local`)

	got, err := HomeDir()
	if err != nil {
		t.Fatalf("HomeDir() error = %v", err)
	}
	want := filepath.Join(`C:\Users\test\AppData\Local`, "cal")
	if got != want {
		t.Fatalf("HomeDir() = %q, want %q", got, want)
	}
}
