package control

import (
	"context"
	"os"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

func TestRunRecordsMissingCapabilityFailure(t *testing.T) {
	svc := newTestService(t)

	run, err := svc.Run(context.Background(), RunRequest{
		CapabilityID: "document.export_pdf",
		Inputs:       map[string]any{},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if run.Status != core.RunStatusFailed || run.Error == nil || run.Error.Code != "capability_not_found" {
		t.Fatalf("Run() = %#v, want persisted capability_not_found failure", run)
	}
	stored, ok, err := svc.GetRun(run.ID)
	if err != nil {
		t.Fatalf("GetRun() error = %v", err)
	}
	if !ok || stored.Error == nil || stored.Error.Code != "capability_not_found" {
		t.Fatalf("stored run = %#v ok %v, want capability_not_found", stored, ok)
	}
}

func TestRunSucceedsWithoutVerify(t *testing.T) {
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
	capabilityID := "document.export_pdf"
	execution := core.Execution{
		Kind: core.ExecutionKindCLI,
		Spec: map[string]any{core.ExecutionSpecArgs: []string{"-test.run=TestRunHelperProcess"}},
	}
	bindingID, err := core.BindingIDForExecution(capabilityID, provider.ID, execution)
	if err != nil {
		t.Fatalf("BindingIDForExecution() error = %v", err)
	}
	capability := core.Capability{
		ID:          capabilityID,
		Description: "Export a document to PDF.",
		Bindings: []core.Binding{{
			ID:           bindingID,
			CapabilityID: capabilityID,
			ProviderID:   provider.ID,
			Execution:    execution,
			Verifier:     &core.Verifier{ID: "verifier_test"},
			Evidence:     []core.EvidenceRef{{ID: "evidence_test"}},
			State:        core.BindingStatePromoted,
		}},
	}
	if err := svc.store.PutCapability(capability); err != nil {
		t.Fatalf("PutCapability() error = %v", err)
	}

	run, err := svc.Run(context.Background(), RunRequest{
		CapabilityID: capabilityID,
		Inputs:       map[string]any{},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if run.Status != core.RunStatusSucceeded || run.BindingID != bindingID || run.ProviderID != provider.ID || run.Verified {
		t.Fatalf("Run() = %#v, want unverified success", run)
	}
}

func TestRunUsesBindingIDConstraint(t *testing.T) {
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
	capabilityID := "document.export_pdf"
	execution := core.Execution{
		Kind: core.ExecutionKindCLI,
		Spec: map[string]any{core.ExecutionSpecArgs: []string{"-test.run=TestRunHelperProcess"}},
	}
	capability := core.Capability{
		ID:          capabilityID,
		Description: "Export a document to PDF.",
		Bindings: []core.Binding{
			{
				ID:           "binding_a",
				CapabilityID: capabilityID,
				ProviderID:   "provider_missing",
				Execution:    execution,
				Verifier:     &core.Verifier{ID: "verifier_test"},
				Evidence:     []core.EvidenceRef{{ID: "evidence_a"}},
				State:        core.BindingStatePromoted,
			},
			{
				ID:           "binding_z",
				CapabilityID: capabilityID,
				ProviderID:   provider.ID,
				Execution:    execution,
				Verifier:     &core.Verifier{ID: "verifier_test"},
				Evidence:     []core.EvidenceRef{{ID: "evidence_z"}},
				State:        core.BindingStatePromoted,
			},
		},
	}
	if err := svc.store.PutCapability(capability); err != nil {
		t.Fatalf("PutCapability() error = %v", err)
	}

	run, err := svc.Run(context.Background(), RunRequest{
		CapabilityID: capabilityID,
		BindingID:    "binding_z",
		Inputs:       map[string]any{},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if run.Status != core.RunStatusSucceeded || run.BindingID != "binding_z" || run.ProviderID != provider.ID {
		t.Fatalf("Run() = %#v, want binding_z success", run)
	}
}

func TestRunRejectsInvalidRequest(t *testing.T) {
	svc := newTestService(t)

	if _, err := svc.Run(context.Background(), RunRequest{Inputs: map[string]any{}}); err == nil {
		t.Fatal("Run() missing capability error = nil, want invalid input")
	}
	if _, err := svc.Run(context.Background(), RunRequest{CapabilityID: "document.export_pdf"}); err == nil {
		t.Fatal("Run() nil inputs error = nil, want invalid input")
	}
}

func TestRunHelperProcess(t *testing.T) {
	if os.Getenv("CALD_RUN_HELPER_PROCESS") != "1" {
		return
	}
	os.Exit(0)
}
