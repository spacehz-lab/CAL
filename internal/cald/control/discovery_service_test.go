package control

import (
	"context"
	"errors"
	"testing"
)

func TestDiscoverRejectsInvalidTargetShape(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Discover(context.Background(), DiscoveryRequest{})
	if err == nil {
		t.Fatal("Discover() error = nil, want invalid target")
	}
	var apiErr APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "invalid_discovery_target" {
		t.Fatalf("Discover() error = %#v, want invalid_discovery_target", err)
	}
}
