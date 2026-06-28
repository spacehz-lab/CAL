package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	e2etest "github.com/spacehz-lab/cal/internal/testsupport/e2e"
)

type e2eBinaries struct {
	repo   string
	dir    string
	calctl string
	cald   string
}

var binaries e2eBinaries

func TestMain(m *testing.M) {
	built, err := buildE2EBinaries()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	binaries = built
	code := m.Run()
	_ = os.RemoveAll(binaries.dir)
	os.Exit(code)
}

func functionalBinaries(t *testing.T) (repo string, calctlBin string, caldBin string) {
	t.Helper()
	if binaries.repo == "" || binaries.calctl == "" || binaries.cald == "" {
		t.Fatal("functional e2e binaries were not built")
	}
	return binaries.repo, binaries.calctl, binaries.cald
}

func buildE2EBinaries() (e2eBinaries, error) {
	wd, err := os.Getwd()
	if err != nil {
		return e2eBinaries{}, fmt.Errorf("get working directory: %w", err)
	}
	repo := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	dir, err := os.MkdirTemp("", "cal-functional-e2e-*")
	if err != nil {
		return e2eBinaries{}, fmt.Errorf("create e2e binary directory: %w", err)
	}
	built := e2eBinaries{
		repo:   repo,
		dir:    dir,
		calctl: filepath.Join(dir, "calctl"),
		cald:   filepath.Join(dir, "cald"),
	}
	if err := buildGoPackage(repo, built.calctl, "./cmd/calctl"); err != nil {
		_ = os.RemoveAll(dir)
		return e2eBinaries{}, err
	}
	if err := buildGoPackage(repo, built.cald, "./cmd/cald"); err != nil {
		_ = os.RemoveAll(dir)
		return e2eBinaries{}, err
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

func addProvider(t *testing.T, repo string, env []string, calctlBin string, providerPath string) e2etest.ProviderSummary {
	t.Helper()
	var provider e2etest.ProviderSummary
	e2etest.RunJSON(t, repo, env, &provider, calctlBin, "providers", "add", "--provider-path", providerPath, "--json")
	return provider
}

func runDiscoveryForProviderPath(t *testing.T, repo string, env []string, calctlBin string, providerPath string, output any, args ...string) e2etest.ProviderSummary {
	t.Helper()
	provider := addProvider(t, repo, env, calctlBin, providerPath)
	command := append([]string{"discovery", "run", "--provider-id", provider.ID}, args...)
	e2etest.RunJSON(t, repo, env, output, calctlBin, command...)
	return provider
}

func runFailDiscoveryForProviderPath(t *testing.T, repo string, env []string, calctlBin string, providerPath string, output any, args ...string) e2etest.ProviderSummary {
	t.Helper()
	provider := addProvider(t, repo, env, calctlBin, providerPath)
	command := append([]string{"discovery", "run", "--provider-id", provider.ID}, args...)
	e2etest.RunFailJSON(t, repo, env, output, calctlBin, command...)
	return provider
}
