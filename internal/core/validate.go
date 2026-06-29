package core

import (
	"fmt"
	"regexp"
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
		if binding.Verify == nil {
			return fmt.Errorf("promoted binding requires verify spec")
		}
		if len(binding.Evidence) == 0 {
			return fmt.Errorf("promoted binding requires evidence")
		}
	}
	if binding.Verify != nil {
		if err := ValidateVerifySpec(*binding.Verify); err != nil {
			return err
		}
	}
	for _, evidence := range binding.Evidence {
		if strings.TrimSpace(evidence.ID) == "" {
			return fmt.Errorf("evidence id is required")
		}
	}
	return nil
}

// ValidateVerifySpec checks deterministic verify spec invariants.
func ValidateVerifySpec(verify VerifySpec) error {
	if !validVerifyLevel(verify.Level) {
		return fmt.Errorf("verify level %q is invalid", verify.Level)
	}
	if !validVerifyMethod(verify.Method) {
		return fmt.Errorf("verify method %q is invalid", verify.Method)
	}
	if verify.Method == VerifyMethodContract && VerifyLevelRank(verify.Level) > VerifyLevelRank(VerifyLevelL1) {
		return fmt.Errorf("contract verification cannot exceed L1")
	}
	if verify.Method == VerifyMethodContract && len(verify.Checks) > 0 {
		return fmt.Errorf("contract verification cannot include checks")
	}
	if verify.Method == VerifyMethodExecute && verify.Level != VerifyLevelL0 && len(verify.Checks) == 0 {
		return fmt.Errorf("verify checks are required for execute method")
	}
	for _, check := range verify.Checks {
		if strings.TrimSpace(check.Subject) == "" {
			return fmt.Errorf("verify check subject is required")
		}
		if !validVerifyPredicate(check.Predicate) {
			return fmt.Errorf("verify predicate %q is invalid", check.Predicate)
		}
		if err := validateVerifyCheckParams(check); err != nil {
			return err
		}
	}
	return nil
}

func validateVerifyCheckParams(check VerifyCheck) error {
	switch check.Predicate {
	case VerifyPredicateEquals, VerifyPredicateNotEquals:
		if _, ok := check.Params["value"]; !ok {
			return fmt.Errorf("verify predicate %s requires params.value", check.Predicate)
		}
	case VerifyPredicateContains:
		if stringParam(check.Params, "value") == "" {
			return fmt.Errorf("verify predicate %s requires params.value", check.Predicate)
		}
	case VerifyPredicateContainsAny:
		if len(stringListParam(check.Params, "values")) == 0 {
			return fmt.Errorf("verify predicate %s requires params.values", check.Predicate)
		}
	case VerifyPredicateRegex:
		pattern := stringParam(check.Params, "pattern")
		if pattern == "" {
			return fmt.Errorf("verify predicate %s requires params.pattern", check.Predicate)
		}
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("verify regex pattern is invalid: %w", err)
		}
	case VerifyPredicateFormat:
		if stringParam(check.Params, "format") == "" {
			return fmt.Errorf("verify predicate %s requires params.format", check.Predicate)
		}
	case VerifyPredicateBytesEqualTransform:
		if stringParam(check.Params, "source") == "" || stringParam(check.Params, "transform") == "" {
			return fmt.Errorf("verify predicate %s requires params.source and params.transform", check.Predicate)
		}
	case VerifyPredicateHashLineMatches:
		if stringParam(check.Params, "source") == "" || stringParam(check.Params, "algorithm") == "" {
			return fmt.Errorf("verify predicate %s requires params.source and params.algorithm", check.Predicate)
		}
	}
	return nil
}

func stringParam(params map[string]any, key string) string {
	if len(params) == 0 {
		return ""
	}
	value, ok := params[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func stringListParam(params map[string]any, key string) []string {
	value, ok := params[key]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		values := make([]string, 0, len(typed))
		for _, value := range typed {
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				values = append(values, trimmed)
			}
		}
		return values
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(fmt.Sprint(item)); text != "" {
				values = append(values, text)
			}
		}
		return values
	default:
		return nil
	}
}

func validVerifyLevel(level VerifyLevel) bool {
	switch level {
	case VerifyLevelL0, VerifyLevelL1, VerifyLevelL2, VerifyLevelL3:
		return true
	default:
		return false
	}
}

func validVerifyMethod(method VerifyMethod) bool {
	switch method {
	case VerifyMethodExecute, VerifyMethodContract:
		return true
	default:
		return false
	}
}

func validVerifyPredicate(predicate VerifyPredicate) bool {
	switch predicate {
	case VerifyPredicateEquals,
		VerifyPredicateNotEquals,
		VerifyPredicateExists,
		VerifyPredicateNonEmpty,
		VerifyPredicateFormat,
		VerifyPredicateContains,
		VerifyPredicateContainsAny,
		VerifyPredicateRegex,
		VerifyPredicateBytesEqualTransform,
		VerifyPredicateHashLineMatches:
		return true
	default:
		return false
	}
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
