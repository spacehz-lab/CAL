package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

func TestObserverCapturesCLIHelpObservation(t *testing.T) {
	path := writeProviderScript(t, "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then echo 'Usage: observed [options]'; exit 0; fi\nexit 64\n")
	result, err := (Observer{}).Observe(context.Background(), core.Provider{
		ID:   "provider_cli",
		Kind: core.ProviderKindCLI,
		Path: path,
	})
	if err != nil {
		t.Fatalf("Observe() error = %v", err)
	}
	if result.ProviderID != "provider_cli" || len(result.Observations) != 1 {
		t.Fatalf("result = %#v, want one observation", result)
	}
	observation := result.Observations[0]
	if observation.Type != "cli_output" || observation.Source != "help" || !strings.Contains(observation.Content["text"].(string), "observed") {
		t.Fatalf("observation = %#v, want help output observation", observation)
	}
}

func TestObserverCapturesUsageFallbackObservation(t *testing.T) {
	path := writeProviderScript(t, "#!/bin/sh\nif [ \"$#\" -eq 0 ]; then echo 'Usage: provider'; exit 0; fi\nexit 9\n")
	result, err := (Observer{}).Observe(context.Background(), core.Provider{
		ID:   "provider_cli",
		Kind: core.ProviderKindCLI,
		Path: path,
	})
	if err != nil {
		t.Fatalf("Observe() error = %v", err)
	}
	if result.ProviderID != "provider_cli" || len(result.Observations) != 1 {
		t.Fatalf("result = %#v, want one usage observation", result)
	}
	observation := result.Observations[0]
	if observation.Type != "cli_output" || observation.Source != "usage" || !strings.Contains(observation.Content["text"].(string), "Usage: provider") {
		t.Fatalf("observation = %#v, want usage output observation", observation)
	}
}

func TestObserverIgnoresNonCLIProvider(t *testing.T) {
	result, err := (Observer{}).Observe(context.Background(), core.Provider{
		ID:   "provider_app",
		Kind: core.ProviderKindApp,
		Path: "/Applications/Fake.app",
	})
	if err != nil {
		t.Fatalf("Observe() error = %v", err)
	}
	if result.ProviderID != "provider_app" || len(result.Observations) != 0 {
		t.Fatalf("result = %#v, want empty non-cli result", result)
	}
}

func writeProviderScript(t *testing.T, script string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "provider")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write provider script: %v", err)
	}
	return path
}
