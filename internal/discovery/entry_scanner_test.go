package discovery

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

func TestScanEntriesFindsEntryExecutable(t *testing.T) {
	binDir := t.TempDir()
	providerPath := filepath.Join(binDir, "fake-exporter")
	writeFakeExecutable(t, providerPath)

	providers, err := ScanEntries(context.Background(), EntryOptions{
		Paths: []string{binDir},
	})
	if err != nil {
		t.Fatalf("ScanEntries() error = %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("ScanEntries() len = %d, want 1: %#v", len(providers), providers)
	}
	wantPath, err := filepath.Abs(providerPath)
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	if providers[0].ID != core.ProviderID(runtime.GOOS, core.ProviderKindCLI, wantPath) || providers[0].Kind != core.ProviderKindCLI || providers[0].Path != wantPath || providers[0].Name != "fake-exporter" {
		t.Fatalf("ScanEntries()[0] = %#v, want entry-scoped executable provider", providers[0])
	}
}

func TestScanEntriesFindsExecutableFromPATH(t *testing.T) {
	binDir := t.TempDir()
	providerPath := filepath.Join(binDir, "path-tool")
	writeFakeExecutable(t, providerPath)
	t.Setenv("PATH", binDir)

	providers, err := ScanEntries(context.Background(), EntryOptions{
		Paths: []string{"PATH"},
	})
	if err != nil {
		t.Fatalf("ScanEntries() error = %v", err)
	}
	if len(providers) != 1 || providers[0].Path != providerPath {
		t.Fatalf("ScanEntries() = %#v, want PATH provider", providers)
	}
}

func TestScanEntriesFindsExplicitEntryPath(t *testing.T) {
	binDir := t.TempDir()
	providerPath := filepath.Join(binDir, "custom-tool")
	writeFakeExecutable(t, providerPath)

	providers, err := ScanEntries(context.Background(), EntryOptions{
		Entries: []string{providerPath},
	})
	if err != nil {
		t.Fatalf("ScanEntries() error = %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("ScanEntries() len = %d, want 1: %#v", len(providers), providers)
	}
	wantPath, err := filepath.Abs(providerPath)
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	if providers[0].ID != core.ProviderID(runtime.GOOS, core.ProviderKindCLI, wantPath) || providers[0].Path != wantPath || providers[0].Name != "custom-tool" {
		t.Fatalf("ScanEntries()[0] = %#v, want explicit entry provider", providers[0])
	}
}

func TestScanEntriesExpandsHomeInEntryPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	binDir := filepath.Join(home, "bin")
	if err := os.Mkdir(binDir, 0o755); err != nil {
		t.Fatalf("create bin dir: %v", err)
	}
	providerPath := filepath.Join(binDir, "home-tool")
	writeFakeExecutable(t, providerPath)

	providers, err := ScanEntries(context.Background(), EntryOptions{
		Entries: []string{"$HOME/bin/home-tool"},
	})
	if err != nil {
		t.Fatalf("ScanEntries() error = %v", err)
	}
	if len(providers) != 1 || providers[0].Path != providerPath {
		t.Fatalf("ScanEntries() = %#v, want expanded home tool", providers)
	}
}

func TestScanEntriesIgnoresMissingAndBlankPaths(t *testing.T) {
	providers, err := ScanEntries(context.Background(), EntryOptions{
		Paths:   []string{"", filepath.Join(t.TempDir(), "missing")},
		Entries: []string{"", filepath.Join(t.TempDir(), "missing-tool")},
	})
	if err != nil {
		t.Fatalf("ScanEntries() error = %v", err)
	}
	if len(providers) != 0 {
		t.Fatalf("ScanEntries() = %#v, want no providers", providers)
	}
}

func writeFakeExecutable(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake executable: %v", err)
	}
}
