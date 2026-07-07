//go:build !windows

package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUsageOutputsCollectsHelp(t *testing.T) {
	path := writeProviderScript(t, "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then echo 'Usage: provider [options]'; exit 0; fi\nexit 64\n")

	outputs, err := UsageOutputs(context.Background(), path)
	if err != nil {
		t.Fatalf("UsageOutputs() error = %v", err)
	}
	if len(outputs) != 1 || outputs[0].Source != sourceHelp || !strings.Contains(outputs[0].Text, "Usage: provider") {
		t.Fatalf("outputs = %#v, want help usage text", outputs)
	}
}

func TestUsageOutputsKeepsNonzeroHelpText(t *testing.T) {
	path := writeProviderScript(t, "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then echo 'Usage: provider [options]'; exit 1; fi\nexit 64\n")

	outputs, err := UsageOutputs(context.Background(), path)
	if err != nil {
		t.Fatalf("UsageOutputs() error = %v", err)
	}
	if len(outputs) != 1 || outputs[0].Source != sourceHelp || !strings.Contains(outputs[0].Text, "Usage: provider") {
		t.Fatalf("outputs = %#v, want nonzero help text", outputs)
	}
}

func TestUsageOutputsFallsBackFromOptionErrorToDashHelp(t *testing.T) {
	path := writeProviderScript(t, "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then echo 'unrecognized option: --help'; exit 0; fi\nif [ \"$1\" = \"-help\" ]; then echo 'Usage: provider [options]'; echo ' -convert fmt'; exit 0; fi\nexit 64\n")

	outputs, err := UsageOutputs(context.Background(), path)
	if err != nil {
		t.Fatalf("UsageOutputs() error = %v", err)
	}
	if len(outputs) != 1 || outputs[0].Source != sourceDashHelp || !strings.Contains(outputs[0].Text, "-convert") {
		t.Fatalf("outputs = %#v, want dash help output", outputs)
	}
}

func TestUsageOutputsCapturesNoArgUsage(t *testing.T) {
	path := writeProviderScript(t, "#!/bin/sh\nif [ \"$#\" -eq 0 ]; then echo 'Usage: provider'; exit 0; fi\nexit 9\n")

	outputs, err := UsageOutputs(context.Background(), path)
	if err != nil {
		t.Fatalf("UsageOutputs() error = %v", err)
	}
	if len(outputs) != 1 || outputs[0].Source != sourceUsage || !strings.Contains(outputs[0].Text, "Usage: provider") {
		t.Fatalf("outputs = %#v, want no-arg usage", outputs)
	}
}

func TestIsUsefulUsageRejectsBareOptionError(t *testing.T) {
	if isUsefulUsage("-z: illegal option") {
		t.Fatal("isUsefulUsage() = true, want bare option error rejected")
	}
}

func TestIsUsefulUsageKeepsOptionErrorWithUsage(t *testing.T) {
	text := "illegal option -- -\nUsage: provider [options]\n -a value\n"
	if !isUsefulUsage(text) {
		t.Fatal("isUsefulUsage() = false, want option error with usage accepted")
	}
}

func TestUsageOutputsReportsCommandFailure(t *testing.T) {
	path := writeProviderScript(t, "#!/bin/sh\nexit 9\n")

	if _, err := UsageOutputs(context.Background(), path); err == nil {
		t.Fatal("UsageOutputs() error = nil, want command failure")
	}
}

func TestCommandOutputTimesOut(t *testing.T) {
	withUsageCommandTimeout(t, 50*time.Millisecond)
	path := writeProviderScript(t, "#!/bin/sh\nexec sleep 10\n")

	if _, err := commandOutput(context.Background(), path, true); err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("commandOutput() error = %v, want timeout", err)
	}
}

func withUsageCommandTimeout(t *testing.T, timeout time.Duration) {
	t.Helper()
	previous := usageCommandTimeout
	usageCommandTimeout = timeout
	t.Cleanup(func() {
		usageCommandTimeout = previous
	})
}

func writeProviderScript(t *testing.T, script string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "provider")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write provider script: %v", err)
	}
	return path
}
