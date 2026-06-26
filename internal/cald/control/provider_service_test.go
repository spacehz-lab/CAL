package control

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

func TestProviderSourcesMutatePathSources(t *testing.T) {
	svc := newTestService(t)

	initial, err := svc.ListSources()
	if err != nil {
		t.Fatalf("ListSources() error = %v", err)
	}
	if len(initial) == 0 {
		t.Fatal("ListSources() returned no default sources")
	}

	cfg, changed, err := svc.AddPathSource("  /tmp/cal-provider-source  ")
	if err != nil {
		t.Fatalf("AddPathSource() error = %v", err)
	}
	if !changed {
		t.Fatal("AddPathSource() changed = false, want true")
	}
	cfg, changed, err = svc.AddPathSource("/tmp/cal-provider-source")
	if err != nil {
		t.Fatalf("AddPathSource() duplicate error = %v", err)
	}
	if changed {
		t.Fatal("AddPathSource() duplicate changed = true, want false")
	}

	cfg, changed, err = svc.RemovePathSource("/tmp/cal-provider-source")
	if err != nil {
		t.Fatalf("RemovePathSource() error = %v", err)
	}
	if !changed {
		t.Fatal("RemovePathSource() changed = false, want true")
	}
	if len(cfg.ProviderSources) != len(initial) {
		t.Fatalf("sources len = %d, want initial len %d", len(cfg.ProviderSources), len(initial))
	}
}

func TestFindProvidersRejectsUnsupportedKind(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.FindProviders(context.Background(), ProviderFindRequest{Kind: "browser"})
	if err == nil {
		t.Fatal("FindProviders() error = nil, want unsupported kind error")
	}
	var apiErr APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "unsupported_provider_kind" {
		t.Fatalf("FindProviders() error = %#v, want unsupported_provider_kind", err)
	}
}

func TestFindProvidersScansConfiguredPathSource(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable-bit provider scan is Unix-specific")
	}
	svc := newTestService(t)
	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "cal-fake-cli")
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake cli: %v", err)
	}
	writeTestConfig(t, svc.Home(), binDir)

	result, err := svc.FindProviders(context.Background(), ProviderFindRequest{Kind: string(core.ProviderKindCLI)})
	if err != nil {
		t.Fatalf("FindProviders() error = %v", err)
	}
	if result.ProvidersCreated != 1 || result.ProvidersUpdated != 0 || len(result.Providers) != 1 {
		t.Fatalf("FindProviders() = %#v, want one created provider", result)
	}
	if result.Providers[0].Name != "cal-fake-cli" || result.Providers[0].Kind != core.ProviderKindCLI {
		t.Fatalf("provider = %#v, want fake cli", result.Providers[0])
	}

	result, err = svc.FindProviders(context.Background(), ProviderFindRequest{Kind: string(core.ProviderKindCLI)})
	if err != nil {
		t.Fatalf("FindProviders() second error = %v", err)
	}
	if result.ProvidersCreated != 0 || result.ProvidersUpdated != 1 {
		t.Fatalf("FindProviders() second counts = created %d updated %d, want 0/1", result.ProvidersCreated, result.ProvidersUpdated)
	}
}

func TestSaveProvidersCountsCreatedAndUpdated(t *testing.T) {
	svc := newTestService(t)
	provider := core.Provider{
		ID:   "provider_test",
		Name: "before",
		Kind: core.ProviderKindCLI,
		Path: "/tmp/provider-test",
	}

	created, updated, err := svc.saveProviders([]core.Provider{provider})
	if err != nil {
		t.Fatalf("saveProviders() error = %v", err)
	}
	if created != 1 || updated != 0 {
		t.Fatalf("saveProviders() counts = %d/%d, want 1/0", created, updated)
	}

	provider.Name = "after"
	created, updated, err = svc.saveProviders([]core.Provider{provider})
	if err != nil {
		t.Fatalf("saveProviders() update error = %v", err)
	}
	if created != 0 || updated != 1 {
		t.Fatalf("saveProviders() update counts = %d/%d, want 0/1", created, updated)
	}
	got, ok, err := svc.GetProvider(provider.ID)
	if err != nil {
		t.Fatalf("GetProvider() error = %v", err)
	}
	if !ok || got.Name != "after" {
		t.Fatalf("GetProvider() = %#v ok %v, want updated provider", got, ok)
	}
}

func writeTestConfig(t *testing.T, home, source string) {
	t.Helper()
	content := `{"provider_sources":[{"kind":"path","value":` + strconv.Quote(source) + `}]}`
	if err := os.WriteFile(filepath.Join(home, "config.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}
