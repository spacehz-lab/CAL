package model

import (
	"strings"
	"testing"
	"time"
)

func TestStableIDsUseShortHashPrefixes(t *testing.T) {
	providerID := ProviderID("darwin", ProviderKindCLI, "/usr/bin/fake")
	if !strings.HasPrefix(providerID, idPrefixProvider) || len(providerID) != len(idPrefixProvider)+idHashLength {
		t.Fatalf("ProviderID() = %q, want provider_ plus short hash", providerID)
	}

	bindingID := BindingID("document.convert", providerID, "cli")
	if !strings.HasPrefix(bindingID, idPrefixBinding) || len(bindingID) != len(idPrefixBinding)+idHashLength {
		t.Fatalf("BindingID() = %q, want binding_ plus short hash", bindingID)
	}
	firstHash := ShortHash("a", "b")
	secondHash := ShortHash("a", "b")
	if firstHash != secondHash {
		t.Fatalf("ShortHash() = %q then %q, want stable value", firstHash, secondHash)
	}
}

func TestBindingIDForExecutionIncludesSpec(t *testing.T) {
	first, err := BindingIDForExecution("document.convert", "provider_cli", Execution{
		Kind: ExecutionKindCLI,
		Spec: map[string]any{ExecutionSpecArgs: []string{"old"}},
	})
	if err != nil {
		t.Fatalf("BindingIDForExecution() error = %v", err)
	}
	second, err := BindingIDForExecution("document.convert", "provider_cli", Execution{
		Kind: ExecutionKindCLI,
		Spec: map[string]any{ExecutionSpecArgs: []string{"new"}},
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
		Spec: map[string]any{ExecutionSpecArgs: []string{"run"}},
	})
	if err != nil {
		t.Fatalf("CanonicalExecution() error = %v", err)
	}
	second, err := CanonicalExecution(Execution{
		Kind: ExecutionKindCLI,
		Spec: map[string]any{ExecutionSpecArgs: []any{"run"}},
	})
	if err != nil {
		t.Fatalf("CanonicalExecution() error = %v", err)
	}
	if first != second {
		t.Fatalf("CanonicalExecution() = %q then %q, want equivalent specs to match", first, second)
	}
}

func TestTraceIDAndRunIDUseUnixNanoUTC(t *testing.T) {
	now := time.Unix(123, 456).In(time.FixedZone("test", 3600))
	if got, want := TraceID(now), idPrefixTrace+"123000000456"; got != want {
		t.Fatalf("TraceID() = %q, want %q", got, want)
	}
	if got, want := RunID(now), idPrefixRun+"123000000456"; got != want {
		t.Fatalf("RunID() = %q, want %q", got, want)
	}
}
