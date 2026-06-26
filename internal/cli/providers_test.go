package cli

import (
	"strings"
	"testing"
)

func TestProvidersSourcesCommands(t *testing.T) {
	home := t.TempDir()
	startCLITestCald(t, home)
	output, err := executeRoot(home, "providers", "sources", "list", "--json")
	if err != nil {
		t.Fatalf("sources list error = %v\n%s", err, output)
	}
	if !strings.Contains(output, `"sources"`) || !strings.Contains(output, `"kind": "path"`) {
		t.Fatalf("sources list output = %q, want JSON provider sources", output)
	}

	output, err = executeRoot(home, "providers", "sources", "add", "--kind", "path", "--value", "/tmp/cal-extra", "--json")
	if err != nil {
		t.Fatalf("sources add error = %v\n%s", err, output)
	}
	if !strings.Contains(output, `"changed": true`) {
		t.Fatalf("sources add output = %q, want changed true", output)
	}

	output, err = executeRoot(home, "providers", "sources", "remove", "--kind", "path", "--value", "/tmp/cal-extra", "--json")
	if err != nil {
		t.Fatalf("sources remove error = %v\n%s", err, output)
	}
	if !strings.Contains(output, `"changed": true`) {
		t.Fatalf("sources remove output = %q, want changed true", output)
	}
}

func TestProvidersSourcesCommandsTextOutput(t *testing.T) {
	home := t.TempDir()
	startCLITestCald(t, home)
	output, err := executeRoot(home, "providers", "sources", "add", "--kind", "path", "--value", "/tmp/cal-text")
	if err != nil {
		t.Fatalf("sources add text error = %v\n%s", err, output)
	}
	if !strings.Contains(output, "/tmp/cal-text") {
		t.Fatalf("sources add text output = %q, want added path", output)
	}

	output, err = executeRoot(home, "providers", "sources", "list")
	if err != nil {
		t.Fatalf("sources list text error = %v\n%s", err, output)
	}
	if !strings.Contains(output, "/tmp/cal-text") {
		t.Fatalf("sources list text output = %q, want added path", output)
	}

	output, err = executeRoot(home, "providers", "sources", "remove", "--kind", "path", "--value", "/tmp/cal-text")
	if err != nil {
		t.Fatalf("sources remove text error = %v\n%s", err, output)
	}
	if !strings.Contains(output, "/tmp/cal-text") {
		t.Fatalf("sources remove text output = %q, want removed path", output)
	}
}
