package core

import (
	"fmt"
	"strings"
)

// ValidateProvider checks Provider invariants.
func ValidateProvider(provider Provider) error {
	if strings.TrimSpace(provider.ID) == "" {
		return fmt.Errorf("provider id is required")
	}
	if !validProviderKind(provider.Kind) {
		return fmt.Errorf("provider kind %q is invalid", provider.Kind)
	}
	if strings.TrimSpace(provider.Path) == "" {
		return fmt.Errorf("provider path is required")
	}
	return nil
}

func validProviderKind(kind ProviderKind) bool {
	switch kind {
	case ProviderKindCLI, ProviderKindApp:
		return true
	default:
		return false
	}
}

// ValidateCapability checks Capability and embedded Binding invariants.
func ValidateCapability(capability Capability) error {
	if strings.TrimSpace(capability.ID) == "" {
		return fmt.Errorf("capability id is required")
	}
	if !ValidCapabilityID(capability.ID) {
		return fmt.Errorf("capability id %q is invalid", capability.ID)
	}
	if hasPromotedBinding(capability.Bindings) && strings.TrimSpace(capability.Description) == "" {
		return fmt.Errorf("promoted capability requires description")
	}
	for _, binding := range capability.Bindings {
		if err := ValidateBinding(capability.ID, binding); err != nil {
			return err
		}
	}
	return nil
}

func hasPromotedBinding(bindings []Binding) bool {
	for _, binding := range bindings {
		if binding.State == BindingStatePromoted {
			return true
		}
	}
	return false
}

// ValidateBinding checks Binding invariants.
func ValidateBinding(ownerCapabilityID string, binding Binding) error {
	if strings.TrimSpace(binding.ID) == "" {
		return fmt.Errorf("binding id is required")
	}
	if strings.TrimSpace(binding.CapabilityID) == "" {
		return fmt.Errorf("binding capability id is required")
	}
	if ownerCapabilityID != "" && binding.CapabilityID != ownerCapabilityID {
		return fmt.Errorf("binding capability id %q does not match owner capability %q", binding.CapabilityID, ownerCapabilityID)
	}
	if strings.TrimSpace(binding.ProviderID) == "" {
		return fmt.Errorf("binding provider id is required")
	}
	if !validBindingState(binding.State) {
		return fmt.Errorf("binding state %q is invalid", binding.State)
	}
	if !validExecutionKind(binding.Execution.Kind) {
		return fmt.Errorf("binding execution kind %q is invalid", binding.Execution.Kind)
	}
	if binding.State == BindingStatePromoted {
		if binding.Verifier == nil {
			return fmt.Errorf("promoted binding requires verifier")
		}
		if len(binding.Evidence) == 0 {
			return fmt.Errorf("promoted binding requires evidence")
		}
	}
	if binding.Verifier != nil && strings.TrimSpace(binding.Verifier.ID) == "" {
		return fmt.Errorf("verifier id is required")
	}
	if binding.Verifier != nil && !ValidVerifierID(binding.Verifier.ID) {
		return fmt.Errorf("verifier id %q is invalid", binding.Verifier.ID)
	}
	for _, evidence := range binding.Evidence {
		if strings.TrimSpace(evidence.ID) == "" {
			return fmt.Errorf("evidence id is required")
		}
	}
	return nil
}

func validBindingState(state BindingState) bool {
	switch state {
	case BindingStatePromoted:
		return true
	default:
		return false
	}
}

func validExecutionKind(kind ExecutionKind) bool {
	switch kind {
	case ExecutionKindCLI, ExecutionKindMenu, ExecutionKindAXAction, ExecutionKindURLOpen:
		return true
	default:
		return false
	}
}

// ValidateRun checks Run invariants.
func ValidateRun(run Run) error {
	if strings.TrimSpace(run.ID) == "" {
		return fmt.Errorf("run id is required")
	}
	if strings.TrimSpace(run.CapabilityID) == "" {
		return fmt.Errorf("run capability id is required")
	}
	switch run.Status {
	case RunStatusSucceeded, RunStatusFailed:
		return nil
	default:
		return fmt.Errorf("run status %q is invalid", run.Status)
	}
}
