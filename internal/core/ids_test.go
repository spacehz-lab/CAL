package core

import (
	"strings"
	"testing"
)

func TestStableIDsUseShortHashPrefixes(t *testing.T) {
	providerID := ProviderID("darwin", ProviderKindCLI, "/usr/bin/fake")
	if !strings.HasPrefix(providerID, "provider_") || len(providerID) != len("provider_")+12 {
		t.Fatalf("ProviderID() = %q, want provider_ plus short hash", providerID)
	}

	bindingID := BindingID("document.export_pdf", providerID, "cli")
	if !strings.HasPrefix(bindingID, "binding_") || len(bindingID) != len("binding_")+12 {
		t.Fatalf("BindingID() = %q, want binding_ plus short hash", bindingID)
	}
	firstHash := ShortHash("a", "b")
	secondHash := ShortHash("a", "b")
	if firstHash != secondHash {
		t.Fatalf("ShortHash() = %q then %q, want stable value", firstHash, secondHash)
	}
}

func TestBindingIDForExecutionIncludesSpec(t *testing.T) {
	first, err := BindingIDForExecution("document.export_pdf", "provider_cli", Execution{
		Kind: ExecutionKindCLI,
		Spec: map[string]any{"args": []string{"old"}},
	})
	if err != nil {
		t.Fatalf("BindingIDForExecution() error = %v", err)
	}
	second, err := BindingIDForExecution("document.export_pdf", "provider_cli", Execution{
		Kind: ExecutionKindCLI,
		Spec: map[string]any{"args": []string{"new"}},
	})
	if err != nil {
		t.Fatalf("BindingIDForExecution() error = %v", err)
	}
	if first == second {
		t.Fatalf("BindingIDForExecution() = %q for different executions", first)
	}
}

func TestCanonicalExecutionIsStableAcrossEquivalentSpecValues(t *testing.T) {
	first, err := CanonicalExecution(Execution{
		Kind: ExecutionKindCLI,
		Spec: map[string]any{"args": []string{"run"}},
	})
	if err != nil {
		t.Fatalf("CanonicalExecution() error = %v", err)
	}
	second, err := CanonicalExecution(Execution{
		Kind: ExecutionKindCLI,
		Spec: map[string]any{"args": []any{"run"}},
	})
	if err != nil {
		t.Fatalf("CanonicalExecution() error = %v", err)
	}
	if first != second {
		t.Fatalf("CanonicalExecution() = %q then %q, want equivalent specs to match", first, second)
	}
}

func TestValidateRunRequiresCoreFields(t *testing.T) {
	valid := Run{ID: "run_abc", CapabilityID: "document.export_pdf", Status: RunStatusSucceeded}
	if err := ValidateRun(valid); err != nil {
		t.Fatalf("ValidateRun() error = %v", err)
	}
	if err := ValidateRun(Run{CapabilityID: "document.export_pdf", Status: RunStatusSucceeded}); err == nil {
		t.Fatal("ValidateRun() error = nil, want missing id error")
	}
	if err := ValidateRun(Run{ID: "run_abc", Status: RunStatusSucceeded}); err == nil {
		t.Fatal("ValidateRun() error = nil, want missing capability id error")
	}
	if err := ValidateRun(Run{ID: "run_abc", CapabilityID: "document.export_pdf", Status: "done"}); err == nil {
		t.Fatal("ValidateRun() error = nil, want invalid status error")
	}
}
