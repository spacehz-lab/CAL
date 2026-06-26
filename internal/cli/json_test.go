package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteJSONCommandError(t *testing.T) {
	var out bytes.Buffer

	err := writeJSONCommandError(&out, commandErrorInvalidRunInput, "invalid input")
	if err == nil {
		t.Fatal("writeJSONCommandError() error = nil, want command error")
	}
	if err.Error() != "invalid input" {
		t.Fatalf("Error() = %q, want invalid input", err.Error())
	}
	if !strings.Contains(out.String(), `"code": "invalid_run_input"`) || !strings.Contains(out.String(), `"message": "invalid input"`) {
		t.Fatalf("output = %q, want JSON command error", out.String())
	}
}
