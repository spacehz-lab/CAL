package cli

import (
	"strings"
	"testing"
)

func TestDaemonCommandsReturnStructuredStatus(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	for _, args := range [][]string{
		{"daemon", "start", "--json"},
		{"daemon", "stop", "--json"},
	} {
		output, err := executeRoot(t.TempDir(), args...)
		if err == nil {
			t.Fatalf("%v succeeded, want structured error", args)
		}
		if !strings.Contains(output, `"error"`) {
			t.Fatalf("%v output = %q, want JSON error", args, output)
		}
	}

	output, err := executeRoot(t.TempDir(), "daemon", "status", "--json")
	if err != nil {
		t.Fatalf("daemon status error = %v\n%s", err, output)
	}
	if !strings.Contains(output, `"running": false`) {
		t.Fatalf("daemon status output = %q, want stopped status", output)
	}
}
