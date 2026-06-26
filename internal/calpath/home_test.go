package calpath

import (
	"path/filepath"
	"testing"
)

func TestHomeDirUsesEnvHome(t *testing.T) {
	home := filepath.Join(t.TempDir(), "env-home")
	t.Setenv(envHome, home)

	got, err := HomeDir()
	if err != nil {
		t.Fatalf("HomeDir() error = %v", err)
	}
	if got != filepath.Clean(home) {
		t.Fatalf("HomeDir() = %q, want %q", got, filepath.Clean(home))
	}
}

func TestWithHomeEnvAddsAndReplacesHome(t *testing.T) {
	first := WithHomeEnv([]string{"A=B"}, "/tmp/one")
	if len(first) != 2 || first[1] != envHome+"="+filepath.Clean("/tmp/one") {
		t.Fatalf("WithHomeEnv(add) = %#v, want appended home", first)
	}

	second := WithHomeEnv(first, "/tmp/two")
	if len(second) != 2 || second[1] != envHome+"="+filepath.Clean("/tmp/two") {
		t.Fatalf("WithHomeEnv(replace) = %#v, want replaced home", second)
	}
}
