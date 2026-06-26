package control

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
	caluse "github.com/spacehz-lab/cal/internal/use"
)

func TestUseSelectsPromotedBindingAndRuns(t *testing.T) {
	svc := newTestService(t)
	t.Setenv("CALD_RUN_HELPER_PROCESS", "1")

	provider := core.Provider{
		ID:   "provider_test_cli",
		Name: "test-cli",
		Kind: core.ProviderKindCLI,
		Path: os.Args[0],
	}
	if err := svc.store.PutProvider(provider); err != nil {
		t.Fatalf("PutProvider() error = %v", err)
	}
	capability := useTestCapability("document.export_pdf", "Export a document to PDF.", provider.ID, core.Execution{
		Kind: core.ExecutionKindCLI,
		Spec: map[string]any{core.ExecutionSpecArgs: []string{"-test.run=TestRunHelperProcess"}},
	})
	if err := svc.store.PutCapability(capability); err != nil {
		t.Fatalf("PutCapability() error = %v", err)
	}

	result, err := svc.Use(context.Background(), caluse.Request{
		Intent: "export this document as pdf",
		Inputs: map[string]any{},
	})
	if err != nil {
		t.Fatalf("Use() error = %v", err)
	}
	if result.Status != core.RunStatusSucceeded || result.Selection == nil || result.Run == nil {
		t.Fatalf("Use() = %#v, want selected successful run", result)
	}
	if result.Selection.CapabilityID != capability.ID || result.Selection.BindingID != "binding_use_test" || result.Run.BindingID != "binding_use_test" {
		t.Fatalf("Use() selection/run = %#v / %#v, want selected binding", result.Selection, result.Run)
	}
}

func TestUseReturnsNoMatch(t *testing.T) {
	svc := newTestService(t)
	capability := useTestCapability("document.export_pdf", "Export a document to PDF.", "provider_test_cli", core.Execution{
		Kind: core.ExecutionKindCLI,
		Spec: map[string]any{core.ExecutionSpecArgs: []string{"export-pdf"}},
	})
	if err := svc.store.PutCapability(capability); err != nil {
		t.Fatalf("PutCapability() error = %v", err)
	}

	result, err := svc.Use(context.Background(), caluse.Request{
		Intent: "resize an image",
		Inputs: map[string]any{},
	})
	if err != nil {
		t.Fatalf("Use() error = %v", err)
	}
	if result.Status != core.RunStatusFailed || result.Error == nil || result.Error.Code != caluse.CodeNoMatch || result.Run != nil {
		t.Fatalf("Use() = %#v, want no_match without run", result)
	}
}

func TestUseReturnsMissingInputs(t *testing.T) {
	svc := newTestService(t)
	capability := useTestCapability("document.export_pdf", "Export a document to PDF.", "provider_test_cli", core.Execution{
		Kind: core.ExecutionKindCLI,
		Spec: map[string]any{core.ExecutionSpecArgs: []string{"export-pdf", "--source", "{{source}}", "--target", "{{target}}"}},
	})
	if err := svc.store.PutCapability(capability); err != nil {
		t.Fatalf("PutCapability() error = %v", err)
	}

	result, err := svc.Use(context.Background(), caluse.Request{
		Intent: "export this document as pdf",
	})
	if err != nil {
		t.Fatalf("Use() error = %v", err)
	}
	if result.Status != core.RunStatusFailed || result.Error == nil || result.Error.Code != caluse.CodeMissingInputs || result.Run != nil {
		t.Fatalf("Use() = %#v, want missing_inputs without run", result)
	}
}

func TestUseFillsTemporaryTargetAndRuns(t *testing.T) {
	svc := newTestService(t)
	providerPath := filepath.Join(t.TempDir(), "writer")
	if err := os.WriteFile(providerPath, []byte(`#!/bin/sh
target=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--target" ]; then
    target="$2"
    break
  fi
  shift
done
printf "ok" > "$target"
`), 0o755); err != nil {
		t.Fatalf("write provider: %v", err)
	}
	provider := core.Provider{
		ID:   "provider_test_cli",
		Name: "test-cli",
		Kind: core.ProviderKindCLI,
		Path: providerPath,
	}
	if err := svc.store.PutProvider(provider); err != nil {
		t.Fatalf("PutProvider() error = %v", err)
	}
	capability := useTestCapability("document.export_pdf", "Export a document to PDF.", provider.ID, core.Execution{
		Kind: core.ExecutionKindCLI,
		Spec: map[string]any{core.ExecutionSpecArgs: []string{"export-pdf", "--source", "{{source}}", "--target", "{{target}}"}},
	})
	if err := svc.store.PutCapability(capability); err != nil {
		t.Fatalf("PutCapability() error = %v", err)
	}

	result, err := svc.Use(context.Background(), caluse.Request{
		Intent: "export this document as pdf",
		Inputs: map[string]any{"source": "input.txt"},
	})
	if err != nil {
		t.Fatalf("Use() error = %v", err)
	}
	if result.Status != core.RunStatusSucceeded || result.Run == nil {
		t.Fatalf("Use() = %#v, want successful run", result)
	}
	target, ok := result.Run.Inputs["target"].(string)
	if !ok || target == "" {
		t.Fatalf("run inputs = %#v, want generated target", result.Run.Inputs)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("generated target missing: %v", err)
	}
}

func TestUseRejectsInvalidRequest(t *testing.T) {
	svc := newTestService(t)
	if _, err := svc.Use(context.Background(), caluse.Request{Inputs: map[string]any{}}); err == nil {
		t.Fatal("Use() missing intent error = nil, want invalid input")
	}
}

func useTestCapability(id, description, providerID string, execution core.Execution) core.Capability {
	return core.Capability{
		ID:          id,
		Description: description,
		Bindings: []core.Binding{{
			ID:           "binding_use_test",
			CapabilityID: id,
			ProviderID:   providerID,
			Execution:    execution,
			Verifier:     &core.Verifier{ID: "verifier_test"},
			Evidence:     []core.EvidenceRef{{ID: "evidence_test"}},
			State:        core.BindingStatePromoted,
		}},
	}
}
