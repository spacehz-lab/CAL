package control

import (
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

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
