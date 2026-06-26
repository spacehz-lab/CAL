package cli

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestDocumentationOutputsCollectsHelp(t *testing.T) {
	path := writeProviderScript(t, "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then echo 'Usage: provider [options]'; exit 0; fi\nexit 64\n")

	outputs, err := DocumentationOutputs(context.Background(), path)
	if err != nil {
		t.Fatalf("DocumentationOutputs() error = %v", err)
	}
	if len(outputs) != 1 || outputs[0].Source != "help" || !strings.Contains(outputs[0].Text, "Usage: provider") {
		t.Fatalf("outputs = %#v, want help text", outputs)
	}
}

func TestDocumentationOutputsKeepsNonzeroHelpText(t *testing.T) {
	path := writeProviderScript(t, "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then echo 'Usage: provider [options]'; exit 1; fi\nexit 64\n")

	outputs, err := DocumentationOutputs(context.Background(), path)
	if err != nil {
		t.Fatalf("DocumentationOutputs() error = %v", err)
	}
	if len(outputs) != 1 || outputs[0].Source != "help" || !strings.Contains(outputs[0].Text, "Usage: provider") {
		t.Fatalf("outputs = %#v, want nonzero help text", outputs)
	}
}

func TestDocumentationOutputsSkipsShortOptionErrorAndUsesDashHelp(t *testing.T) {
	path := writeProviderScript(t, "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then echo 'unrecognized option: --help'; exit 0; fi\nif [ \"$1\" = \"-help\" ]; then echo 'Usage: provider [options]'; echo ' -convert fmt'; exit 0; fi\nexit 64\n")

	outputs, err := DocumentationOutputs(context.Background(), path)
	if err != nil {
		t.Fatalf("DocumentationOutputs() error = %v", err)
	}
	if len(outputs) != 1 || outputs[0].Source != "help" || !strings.Contains(outputs[0].Text, "-convert") {
		t.Fatalf("outputs = %#v, want dash help output", outputs)
	}
}

func TestIsUsefulDocumentationRejectsBareOptionError(t *testing.T) {
	if isUsefulDocumentation("-z: illegal option") {
		t.Fatal("isUsefulDocumentation() = true, want bare option error rejected")
	}
}

func TestIsUsefulDocumentationKeepsOptionErrorWithUsage(t *testing.T) {
	text := "illegal option -- -\nUsage: provider [options]\n -a value\n"
	if !isUsefulDocumentation(text) {
		t.Fatal("isUsefulDocumentation() = false, want option error with usage accepted")
	}
}

func TestDocumentationOutputsReportsCommandFailure(t *testing.T) {
	path := writeProviderScript(t, "#!/bin/sh\nexit 9\n")

	if _, err := DocumentationOutputs(context.Background(), path); err == nil {
		t.Fatal("DocumentationOutputs() error = nil, want command failure")
	}
}

func TestCommandOutputTimesOut(t *testing.T) {
	withCommandTimeout(t, 50*time.Millisecond)
	path := writeProviderScript(t, "#!/bin/sh\nexec sleep 10\n")

	if _, err := commandOutput(context.Background(), path, true); err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("commandOutput() error = %v, want timeout", err)
	}
}

func withCommandTimeout(t *testing.T, timeout time.Duration) {
	t.Helper()
	previous := commandTimeout
	commandTimeout = timeout
	t.Cleanup(func() {
		commandTimeout = previous
	})
}
