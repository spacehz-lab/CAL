package control

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

func newTestService(t *testing.T) Service {
	t.Helper()
	svc, err := NewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return svc
}

func TestNewServiceInitializesStore(t *testing.T) {
	svc := newTestService(t)
	if svc.Home() == "" {
		t.Fatal("Home() = empty, want CAL_HOME")
	}
	for _, dir := range []string{"providers", "capabilities", "discovery", "runs"} {
		if _, err := os.Stat(filepath.Join(svc.Home(), dir)); err != nil {
			t.Fatalf("store dir %q was not initialized: %v", dir, err)
		}
	}
}

func testProviderRecord() core.Provider {
	return core.Provider{
		ID:   "provider_test",
		Name: "test-provider",
		Kind: core.ProviderKindCLI,
		Path: "/tmp/provider-test",
	}
}

func testCapabilityRecord(t *testing.T, providerID string) core.Capability {
	t.Helper()
	capabilityID := "document.export_pdf"
	execution := core.Execution{
		Kind: core.ExecutionKindCLI,
		Spec: map[string]any{core.ExecutionSpecArgs: []string{"--version"}},
	}
	bindingID, err := core.BindingIDForExecution(capabilityID, providerID, execution)
	if err != nil {
		t.Fatalf("BindingIDForExecution() error = %v", err)
	}
	return core.Capability{
		ID:          capabilityID,
		Description: "Export a document to PDF.",
		Bindings: []core.Binding{{
			ID:           bindingID,
			CapabilityID: capabilityID,
			ProviderID:   providerID,
			Execution:    execution,
			Verify:       testVerifySpec(),
			Evidence:     []core.EvidenceRef{{ID: "evidence_test"}},
			State:        core.BindingStatePromoted,
		}},
	}
}

func testVerifySpec() *core.VerifySpec {
	return &core.VerifySpec{
		Level:  core.VerifyLevelL2,
		Method: core.VerifyMethodExecute,
		Checks: []core.VerifyCheck{{Subject: "target", Predicate: core.VerifyPredicateExists}},
	}
}
