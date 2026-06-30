package runtime

import (
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

func TestResolveSelectsPromotedBindingDeterministically(t *testing.T) {
	capability := core.Capability{
		ID: "document.convert",
		Bindings: []core.Binding{
			promotedBinding("binding_z", "provider_z", core.ExecutionKindCLI),
			promotedBinding("binding_a", "provider_a", core.ExecutionKindCLI),
			{
				ID:           "binding_unpromoted",
				CapabilityID: "document.convert",
				ProviderID:   "provider_unpromoted",
				Execution:    core.Execution{Kind: core.ExecutionKindCLI},
			},
		},
	}

	resolution, err := NewRunner(DefaultRegistry()).Resolve(capability, ResolveOptions{Strategy: DefaultStrategy})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolution.Binding.ID != "binding_a" || resolution.BindingsConsidered != 2 {
		t.Fatalf("resolution = %#v, want binding_a from promoted bindings", resolution)
	}
}

func TestResolveFiltersProvider(t *testing.T) {
	capability := core.Capability{
		ID: "document.convert",
		Bindings: []core.Binding{
			promotedBinding("binding_a", "provider_a", core.ExecutionKindCLI),
			promotedBinding("binding_b", "provider_b", core.ExecutionKindCLI),
		},
	}

	resolution, err := NewRunner(DefaultRegistry()).Resolve(capability, ResolveOptions{ProviderID: "provider_b", Strategy: DefaultStrategy})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolution.Binding.ID != "binding_b" || resolution.BindingsConsidered != 1 {
		t.Fatalf("resolution = %#v, want provider_b binding", resolution)
	}
}

func TestResolveFiltersBindingID(t *testing.T) {
	capability := core.Capability{
		ID: "document.convert",
		Bindings: []core.Binding{
			promotedBinding("binding_a", "provider_a", core.ExecutionKindCLI),
			promotedBinding("binding_b", "provider_b", core.ExecutionKindCLI),
		},
	}

	resolution, err := NewRunner(DefaultRegistry()).Resolve(capability, ResolveOptions{BindingID: "binding_b", Strategy: DefaultStrategy})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolution.Binding.ID != "binding_b" || resolution.BindingsConsidered != 1 {
		t.Fatalf("resolution = %#v, want binding_b", resolution)
	}
}

func TestResolveRejectsMissingBindingID(t *testing.T) {
	capability := core.Capability{
		ID: "document.convert",
		Bindings: []core.Binding{
			promotedBinding("binding_a", "provider_a", core.ExecutionKindCLI),
		},
	}

	if _, err := NewRunner(DefaultRegistry()).Resolve(capability, ResolveOptions{BindingID: "binding_missing", Strategy: DefaultStrategy}); err == nil {
		t.Fatal("Resolve() error = nil, want missing binding rejection")
	}
}

func TestResolveRejectsUnsupportedExecutionKind(t *testing.T) {
	capability := core.Capability{
		ID: "document.convert",
		Bindings: []core.Binding{
			promotedBinding("binding_bad", "provider_bad", "direct_file"),
		},
	}

	if _, err := NewRunner(DefaultRegistry()).Resolve(capability, ResolveOptions{Strategy: DefaultStrategy}); err == nil {
		t.Fatal("Resolve() error = nil, want unsupported execution kind rejection")
	}
}

func promotedBinding(id, providerID string, kind core.ExecutionKind) core.Binding {
	return core.Binding{
		ID:           id,
		CapabilityID: "document.convert",
		ProviderID:   providerID,
		Execution:    core.Execution{Kind: kind},
		Verify: &core.VerifySpec{
			Level:  core.VerifyLevelL2,
			Method: core.VerifyMethodExecute,
			Checks: []core.VerifyCheck{{Subject: core.VerifySubject{Type: core.VerifySubjectFile, Input: "target"}, Predicate: core.VerifyPredicateExists}},
		},
		Evidence: []core.EvidenceRef{{ID: "evidence_" + id}},
		State:    core.BindingStatePromoted,
	}
}
