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

func TestDocumentationOutputsFallsBackToManAfterLowSignalHelp(t *testing.T) {
	dir := t.TempDir()
	provider := writeScript(t, dir, "provider", "#!/bin/sh\necho ok\nexit 0\n")
	writeScript(t, dir, "man", "#!/bin/sh\nif [ \"$1\" = \"provider\" ]; then echo 'manual usage text'; exit 0; fi\nexit 2\n")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	outputs, err := DocumentationOutputs(context.Background(), provider)
	if err != nil {
		t.Fatalf("DocumentationOutputs() error = %v", err)
	}
	if len(outputs) != 1 || outputs[0].Source != "man" || !strings.Contains(outputs[0].Text, "manual usage") {
		t.Fatalf("outputs = %#v, want man fallback", outputs)
	}
}

func TestManOutputReturnsText(t *testing.T) {
	dir := t.TempDir()
	provider := writeScript(t, dir, "provider", "#!/bin/sh\nexit 0\n")
	writeScript(t, dir, "man", "#!/bin/sh\nif [ \"$1\" = \"provider\" ]; then echo manual text; exit 0; fi\nexit 2\n")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	output, err := manOutput(context.Background(), provider)
	if err != nil {
		t.Fatalf("manOutput() error = %v", err)
	}
	if strings.TrimSpace(output) != "manual text" {
		t.Fatalf("output = %q, want manual text", output)
	}
}

func TestManOutputRejectsPathMismatch(t *testing.T) {
	dir := t.TempDir()
	provider := filepath.Join(t.TempDir(), "provider")
	if err := os.WriteFile(provider, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write provider: %v", err)
	}
	writeScript(t, dir, "provider", "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if _, err := manOutput(context.Background(), provider); err == nil || !strings.Contains(err.Error(), "not provider path") {
		t.Fatalf("manOutput() error = %v, want path mismatch", err)
	}
}

func TestManOutputReportsEmptyOutput(t *testing.T) {
	dir := t.TempDir()
	provider := writeScript(t, dir, "provider", "#!/bin/sh\nexit 0\n")
	writeScript(t, dir, "man", "#!/bin/sh\nexit 0\n")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if _, err := manOutput(context.Background(), provider); err == nil || !strings.Contains(err.Error(), "empty output") {
		t.Fatalf("manOutput() error = %v, want empty output", err)
	}
}

func TestManOutputTimesOut(t *testing.T) {
	withCommandTimeout(t, 50*time.Millisecond)
	dir := t.TempDir()
	provider := writeScript(t, dir, "provider", "#!/bin/sh\nexit 0\n")
	writeScript(t, dir, "man", "#!/bin/sh\nexec sleep 10\n")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if _, err := manOutput(context.Background(), provider); err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("manOutput() error = %v, want timeout", err)
	}
}

func writeScript(t *testing.T, dir, name, script string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write script %s: %v", name, err)
	}
	return path
}
