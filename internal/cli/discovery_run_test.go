package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

func TestRunDiscoverProviderRequiresExactlyOneTarget(t *testing.T) {
	for _, opts := range []discoveryRunOptions{
		{},
		{providerPath: "/tmp/provider", providerID: "provider_test"},
	} {
		_, err := buildDiscoveryRunRequest(opts)
		commandErr, ok := err.(commandError)
		if !ok || commandErr.Code != string(commandErrorInvalidDiscoveryTarget) {
			t.Fatalf("buildDiscoveryRunRequest(%#v) error = %#v, want invalid_discovery_target", opts, err)
		}
	}
}

func TestDiscoverProviderCommandJSONResultAndError(t *testing.T) {
	home := t.TempDir()
	startCLITestCald(t, home)
	installCLITestVerifier(t, home, "file_parse_pdf", pdfMagicVerifierScript())
	providerPath := writeAcquisitionScript(t)

	output, err := executeRoot(home, "discovery", "run", "--provider-path", providerPath, "--mode", "rules", "--json")
	if err != nil {
		t.Fatalf("discovery run json error = %v\n%s", err, output)
	}
	if !strings.Contains(output, `"type": "provider_path"`) || !strings.Contains(output, `"bindings_promoted": 1`) {
		t.Fatalf("discovery run json output = %q, want provider_path acquisition", output)
	}

	output, err = executeRoot(home, "discovery", "run", "--provider-path", filepath.Join(home, "missing"), "--json")
	if err == nil {
		t.Fatalf("discovery run missing path succeeded, want error\n%s", output)
	}
	if !strings.Contains(output, `"code": "target_provider_not_found"`) || !strings.Contains(output, "provider path") {
		t.Fatalf("discovery run error output = %q, want provider-path specific error", output)
	}

	output, err = executeRoot(home, "discovery", "run", "--provider-path", home, "--json")
	if err == nil {
		t.Fatalf("discovery run directory path succeeded, want error\n%s", output)
	}
	if !strings.Contains(output, `"code": "target_provider_not_found"`) {
		t.Fatalf("discovery run directory error output = %q, want target_provider_not_found", output)
	}
}

func TestDiscoverProviderCommandByID(t *testing.T) {
	home := t.TempDir()
	startCLITestCald(t, home)
	installCLITestVerifier(t, home, "file_parse_pdf", pdfMagicVerifierScript())
	store := newTestStoreWithHome(t, home)
	if err := store.PutProvider(testCLIProvider("provider_cli", writeAcquisitionScript(t))); err != nil {
		t.Fatalf("PutProvider() error = %v", err)
	}

	output, err := executeRoot(home, "discovery", "run", "--provider-id", "provider_cli", "--mode", "rules", "--json")
	if err != nil {
		t.Fatalf("discovery run --provider-id error = %v\n%s", err, output)
	}
	if !strings.Contains(output, `"type": "provider"`) || !strings.Contains(output, `"bindings_promoted": 1`) {
		t.Fatalf("discovery run --provider-id output = %q, want provider acquisition", output)
	}
}

func TestDiscoverProviderCommandHidesModeFlag(t *testing.T) {
	output, err := executeRoot(t.TempDir(), "discovery", "run", "--help")
	if err != nil {
		t.Fatalf("discovery run help error = %v\n%s", err, output)
	}
	if strings.Contains(output, "--mode") {
		t.Fatalf("discovery run help = %q, want hidden mode flag omitted", output)
	}
}

func testCLIProvider(id, path string) core.Provider {
	return core.Provider{
		ID:   id,
		Name: "fake",
		Kind: core.ProviderKindCLI,
		Path: path,
	}
}
