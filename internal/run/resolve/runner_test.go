package resolve

import (
	"errors"
	"reflect"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestRunSelectsBindingByID(t *testing.T) {
	capability := capabilityWithBindings(
		binding("binding_a", "provider_a", []string{"run-a"}),
		binding("binding_b", "provider_b", []string{"run-b"}),
	)
	result, err := NewRunner().Run(&Request{Capability: capability, BindingID: "binding_b", Inputs: map[string]any{}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Binding.ID != "binding_b" {
		t.Fatalf("binding id = %q, want binding_b", result.Binding.ID)
	}
}

func TestRunSelectsBindingByProviderID(t *testing.T) {
	capability := capabilityWithBindings(
		binding("binding_a", "provider_a", []string{"run-a"}),
		binding("binding_b", "provider_b", []string{"run-b"}),
	)
	result, err := NewRunner().Run(&Request{Capability: capability, ProviderID: "provider_b", Inputs: map[string]any{}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Binding.ProviderID != "provider_b" {
		t.Fatalf("provider id = %q, want provider_b", result.Binding.ProviderID)
	}
}

func TestRunIgnoresNonPromotedBindings(t *testing.T) {
	capability := capabilityWithBindings(
		model.Binding{
			ID:           "binding_draft",
			CapabilityID: "capability_test",
			ProviderID:   "provider_a",
			Execution:    model.Execution{Kind: model.ExecutionKindCLI, Spec: map[string]any{model.ExecutionSpecArgs: []string{"draft"}}},
		},
		binding("binding_promoted", "provider_b", []string{"run"}),
	)
	result, err := NewRunner().Run(&Request{Capability: capability, Inputs: map[string]any{}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Binding.ID != "binding_promoted" {
		t.Fatalf("binding id = %q, want binding_promoted", result.Binding.ID)
	}
}

func TestRunPrefersSatisfiedInputs(t *testing.T) {
	capability := capabilityWithBindings(
		binding("binding_source_target", "provider_a", []string{"run", "{{source}}", "{{target}}"}),
		binding("binding_source", "provider_b", []string{"run", "{{source}}"}),
	)
	result, err := NewRunner().Run(&Request{Capability: capability, Inputs: map[string]any{"source": "input.txt"}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Binding.ID != "binding_source" {
		t.Fatalf("binding id = %q, want binding_source", result.Binding.ID)
	}
}

func TestRunReturnsRequiredInputs(t *testing.T) {
	capability := capabilityWithBindings(binding("binding_a", "provider_a", []string{"run", "{{target}}", "{{source}}"}))
	result, err := NewRunner().Run(&Request{Capability: capability, Inputs: map[string]any{}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	want := []string{"source", "target"}
	if !reflect.DeepEqual(result.RequiredInputs, want) {
		t.Fatalf("required inputs = %#v, want %#v", result.RequiredInputs, want)
	}
}

func TestRunFiltersByMinVerifyLevel(t *testing.T) {
	capability := capabilityWithBindings(
		bindingWithVerify("binding_l1", "provider_a", model.VerifyLevelL1),
		bindingWithVerify("binding_l2", "provider_b", model.VerifyLevelL2),
	)
	result, err := NewRunner().Run(&Request{Capability: capability, Inputs: map[string]any{}, MinVerifyLevel: model.VerifyLevelL2})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Binding.ID != "binding_l2" {
		t.Fatalf("binding id = %q, want binding_l2", result.Binding.ID)
	}
}

func TestRunReturnsBindingNotFound(t *testing.T) {
	capability := capabilityWithBindings(binding("binding_a", "provider_a", []string{"run"}))
	_, err := NewRunner().Run(&Request{Capability: capability, BindingID: "missing", Inputs: map[string]any{}})
	if !errors.Is(err, ErrBindingNotFound) {
		t.Fatalf("Run() error = %v, want ErrBindingNotFound", err)
	}
}

func capabilityWithBindings(bindings ...model.Binding) *model.Capability {
	return &model.Capability{ID: "capability_test", Description: "test capability", Bindings: bindings}
}

func binding(id string, providerID string, args []string) model.Binding {
	return model.Binding{
		ID:           id,
		CapabilityID: "capability_test",
		ProviderID:   providerID,
		Execution:    model.Execution{Kind: model.ExecutionKindCLI, Spec: map[string]any{model.ExecutionSpecArgs: args}},
		Verify:       executeVerifySpec(model.VerifyLevelL1),
		Evidence:     []model.EvidenceRef{{ID: "evidence_1"}},
		State:        model.BindingStatePromoted,
	}
}

func bindingWithVerify(id string, providerID string, level model.VerifyLevel) model.Binding {
	item := binding(id, providerID, []string{"run"})
	item.Verify = executeVerifySpec(level)
	return item
}

func executeVerifySpec(level model.VerifyLevel) *model.VerifySpec {
	return &model.VerifySpec{
		Level:  level,
		Method: model.VerifyMethodExecute,
		Checks: []model.VerifyCheck{{
			Subject:   model.VerifySubject{Type: model.VerifySubjectStdout},
			Predicate: model.VerifyPredicateNonEmpty,
		}},
	}
}
