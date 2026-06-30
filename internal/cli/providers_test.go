package cli

import (
	"strings"
	"testing"
)

func TestProvidersHelpHidesSources(t *testing.T) {
	output, err := executeRoot(t.TempDir(), "providers", "--help")
	if err != nil {
		t.Fatalf("providers help error = %v\n%s", err, output)
	}
	if strings.Contains(output, "sources") {
		t.Fatalf("providers help = %q, want sources hidden", output)
	}
}

func TestProvidersAddAndGetByPath(t *testing.T) {
	home := t.TempDir()
	startCLITestCald(t, home)
	providerPath := writeAcquisitionScript(t)

	output, err := executeRoot(home, "providers", "add", "--provider-path", providerPath, "--json")
	if err != nil {
		t.Fatalf("providers add error = %v\n%s", err, output)
	}
	if !strings.Contains(output, `"path": `) || !strings.Contains(output, providerPath) {
		t.Fatalf("providers add output = %q, want provider path", output)
	}

	output, err = executeRoot(home, "providers", "get", "--provider-path", providerPath, "--json")
	if err != nil {
		t.Fatalf("providers get by path error = %v\n%s", err, output)
	}
	if !strings.Contains(output, `"path": `) || !strings.Contains(output, providerPath) {
		t.Fatalf("providers get by path output = %q, want provider path", output)
	}
}
