package control

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestDiscoverRejectsInvalidTargetShape(t *testing.T) {
	svc := newTestService(t)

	for _, req := range []DiscoveryRequest{
		{},
		{ProviderID: "provider_test", ProviderPath: "/tmp/provider-test"},
	} {
		_, err := svc.Discover(context.Background(), req)
		if err == nil {
			t.Fatalf("Discover(%#v) error = nil, want invalid target", req)
		}
		var apiErr APIError
		if !errors.As(err, &apiErr) || apiErr.Code != "invalid_discovery_target" {
			t.Fatalf("Discover(%#v) error = %#v, want invalid_discovery_target", req, err)
		}
	}
}

func TestDiscoverRejectsUnknownProviderPath(t *testing.T) {
	svc := newTestService(t)

	for _, providerPath := range []string{
		filepath.Join(t.TempDir(), "missing-cli"),
		t.TempDir(),
	} {
		_, err := svc.Discover(context.Background(), DiscoveryRequest{
			ProviderPath: providerPath,
		})
		if err == nil {
			t.Fatalf("Discover(%q) error = nil, want target_provider_not_found", providerPath)
		}
		var apiErr APIError
		if !errors.As(err, &apiErr) || apiErr.Code != "target_provider_not_found" {
			t.Fatalf("Discover(%q) error = %#v, want target_provider_not_found", providerPath, err)
		}
	}
}
