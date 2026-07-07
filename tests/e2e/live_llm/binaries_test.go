package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

type liveLLMBinaries struct {
	repo   string
	dir    string
	calctl string
	cald   string
}

var binaries liveLLMBinaries

func TestMain(m *testing.M) {
	if !liveLLMEnabled() {
		os.Exit(m.Run())
	}
	built, err := buildLiveLLMBinaries()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	binaries = built
	code := m.Run()
	_ = os.RemoveAll(binaries.dir)
	os.Exit(code)
}

func liveBinaries(t *testing.T) (repo string, calctlBin string, caldBin string) {
	t.Helper()
	if !liveLLMEnabled() {
		t.Skip("set CAL_LIVE_LLM_E2E=1 and CAL_LLM_* to run live LLM e2e")
	}
	if binaries.repo == "" || binaries.calctl == "" || binaries.cald == "" {
		t.Fatal("live LLM e2e binaries were not built")
	}
	return binaries.repo, binaries.calctl, binaries.cald
}

func buildLiveLLMBinaries() (liveLLMBinaries, error) {
	wd, err := os.Getwd()
	if err != nil {
		return liveLLMBinaries{}, fmt.Errorf("get working directory: %w", err)
	}
	repo := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	dir, err := os.MkdirTemp("", "cal-live-llm-e2e-*")
	if err != nil {
		return liveLLMBinaries{}, fmt.Errorf("create live LLM binary directory: %w", err)
	}
	built := liveLLMBinaries{
		repo:   repo,
		dir:    dir,
		calctl: filepath.Join(dir, "calctl"),
		cald:   filepath.Join(dir, "cald"),
	}
	if err := buildGoPackage(repo, built.calctl, "./cmd/calctl"); err != nil {
		_ = os.RemoveAll(dir)
		return liveLLMBinaries{}, err
	}
	if err := buildGoPackage(repo, built.cald, "./cmd/cald"); err != nil {
		_ = os.RemoveAll(dir)
		return liveLLMBinaries{}, err
	}
	return built, nil
}

func buildGoPackage(repo string, output string, pkg string) error {
	cmd := exec.Command("go", "build", "-o", output, pkg)
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build %s failed: %w\n%s", pkg, err, out)
	}
	return nil
}
