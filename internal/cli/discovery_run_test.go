package cli

import (
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

func TestRunDiscoverProviderRequiresExactlyOneTarget(t *testing.T) {
	_, err := buildDiscoveryRunRequest(discoveryRunOptions{})
	commandErr, ok := err.(commandError)
	if !ok || commandErr.Code != string(commandErrorInvalidDiscoveryTarget) {
		t.Fatalf("buildDiscoveryRunRequest() error = %#v, want invalid_discovery_target", err)
	}
}

func TestDiscoverProviderCommandRequiresProviderID(t *testing.T) {
	home := t.TempDir()
	startCLITestCald(t, home)

	output, err := executeRoot(home, "discovery", "run", "--json")
	if err == nil {
		t.Fatalf("discovery run without provider-id succeeded, want error\n%s", output)
	}
	if !strings.Contains(output, `"code": "invalid_discovery_target"`) {
		t.Fatalf("discovery run error output = %q, want invalid_discovery_target", output)
	}
}

func TestDiscoverProviderCommandByID(t *testing.T) {
	home := t.TempDir()
	startCLITestCald(t, home)
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
