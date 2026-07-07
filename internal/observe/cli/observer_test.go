//go:build !windows

package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/observe"
)

func TestObserverCapturesCLIUsageObservation(t *testing.T) {
	path := writeProviderScript(t, "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then echo 'Usage: observed [options]'; exit 0; fi\nexit 64\n")

	result, err := NewObserver().Observe(context.Background(), &observe.Request{Provider: &model.Provider{
		ID:   "provider_cli",
		Kind: model.ProviderKindCLI,
		Path: path,
	}})
	if err != nil {
		t.Fatalf("Observe() error = %v", err)
	}
	if result.ProviderID != "provider_cli" || len(result.Observations) != 1 {
		t.Fatalf("result = %#v, want one observation", result)
	}
	observation := result.Observations[0]
	if observation.Type != observe.ObservationTypeCLIOutput || observation.Source != sourceHelp {
		t.Fatalf("observation = %#v, want help CLI output", observation)
	}
	text, ok := observation.Content[observe.ObservationContentText].(string)
	if !ok || !strings.Contains(text, "observed") {
		t.Fatalf("content = %#v, want observed text", observation.Content)
	}
}

func TestObserverReturnsEmptyForNonCLIProvider(t *testing.T) {
	result, err := NewObserver().Observe(context.Background(), &observe.Request{Provider: &model.Provider{
		ID:   "provider_app",
		Kind: model.ProviderKindApp,
		Path: "/Applications/Fake.app",
	}})
	if err != nil {
		t.Fatalf("Observe() error = %v", err)
	}
	if result.ProviderID != "provider_app" || len(result.Observations) != 0 {
		t.Fatalf("result = %#v, want empty non-CLI result", result)
	}
}
