//go:build darwin

package discovery

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

func TestScanEntriesFindsExplicitAppBundle(t *testing.T) {
	appDir := t.TempDir()
	appPath := filepath.Join(appDir, "Document Tool.app")
	if err := os.Mkdir(appPath, 0o755); err != nil {
		t.Fatalf("create fake app bundle: %v", err)
	}

	providers, err := ScanEntries(context.Background(), EntryOptions{
		Entries: []string{appPath},
	})
	if err != nil {
		t.Fatalf("ScanEntries() error = %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("ScanEntries() len = %d, want 1: %#v", len(providers), providers)
	}
	wantPath, err := filepath.Abs(appPath)
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	if providers[0].ID != core.ProviderID(runtime.GOOS, core.ProviderKindApp, wantPath) || providers[0].Kind != core.ProviderKindApp || providers[0].Path != wantPath || providers[0].Name != "Document Tool" {
		t.Fatalf("ScanEntries()[0] = %#v, want explicit app entry", providers[0])
	}
}

func TestScanEntriesDoesNotMergeCLIAndAppEntries(t *testing.T) {
	dir := t.TempDir()
	writeFakeExecutable(t, filepath.Join(dir, "document-tool"))
	if err := os.Mkdir(filepath.Join(dir, "Document Tool.app"), 0o755); err != nil {
		t.Fatalf("create fake app bundle: %v", err)
	}

	providers, err := ScanEntries(context.Background(), EntryOptions{
		Entries: []string{
			filepath.Join(dir, "document-tool"),
			filepath.Join(dir, "Document Tool.app"),
		},
	})
	if err != nil {
		t.Fatalf("ScanEntries() error = %v", err)
	}
	if len(providers) != 2 {
		t.Fatalf("ScanEntries() len = %d, want 2: %#v", len(providers), providers)
	}
	if providers[0].ID == providers[1].ID {
		t.Fatalf("ScanEntries() merged separate entries: %#v", providers)
	}
}
