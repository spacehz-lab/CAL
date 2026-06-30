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
	if verify.Method == VerifyMethodExecute && verify.Level != VerifyLevelL0 && len(verify.Checks) == 0 {
		return fmt.Errorf("verify checks are required for execute method")
	}
	if verify.Method != VerifyMethodExecute {
		return nil
	}
	rules := verifySubjectRuleMap()
	for _, check := range verify.Checks {
		if err := validateVerifyCheck(check, rules); err != nil {
			return err
		}
	}
	return nil
}

func validateVerifyCheck(check VerifyCheck, rules map[VerifySubjectType]VerifySubjectRule) error {
	rule, ok := rules[check.Subject.Type]
	if !ok {
		return fmt.Errorf("verify subject type %q is invalid", check.Subject.Type)
	}
	if rule.RequiresInput && strings.TrimSpace(check.Subject.Input) == "" {
		return fmt.Errorf("verify subject %s requires input", check.Subject.Type)
	}
	if !rule.RequiresInput && strings.TrimSpace(check.Subject.Input) != "" {
		return fmt.Errorf("verify subject %s cannot include input", check.Subject.Type)
	}
	if !verifyPredicateAllowed(rule, check.Predicate) {
		return fmt.Errorf("verify predicate %s is invalid for subject %s", check.Predicate, check.Subject.Type)
	}
	if err := validateVerifyCheckParams(check, rule); err != nil {
		return err
	}
	if check.Predicate == VerifyPredicateRegex {
		pattern := stringParam(check.Params, "pattern")
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("verify regex pattern is invalid: %w", err)
		}
	}
	return nil
}

func validateVerifyCheckParams(check VerifyCheck, rule VerifySubjectRule) error {
	paramRules := rule.ParamRules[check.Predicate]
	if len(paramRules) == 0 {
		paramRules = map[string]VerifyParamRule{}
		for _, key := range rule.RequiredParams[check.Predicate] {
			paramRules[key] = VerifyParamRule{Required: true}
		}
	}
	for key, paramRule := range paramRules {
		if paramRule.Required {
			switch key {
			case "values":
				if len(stringListParam(check.Params, key)) == 0 {
					return fmt.Errorf("verify predicate %s requires params.%s", check.Predicate, key)
				}
			default:
				if _, ok := check.Params[key]; !ok {
					return fmt.Errorf("verify predicate %s requires params.%s", check.Predicate, key)
				}
			}
		}
		if len(paramRule.AllowedValues) > 0 {
			value := stringParam(check.Params, key)
			if value == "" || !verifyParamValueAllowed(value, paramRule.AllowedValues) {
				return fmt.Errorf("verify predicate %s params.%s %q is not supported", check.Predicate, key, value)
			}
		}
	}
	return nil
}

func verifyParamValueAllowed(value string, allowedValues []string) bool {
	for _, allowed := range allowedValues {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(allowed)) {
			return true
		}
	}
	return false
}

func verifySubjectRuleMap() map[VerifySubjectType]VerifySubjectRule {
	rules := VerifySubjectRules()
	byType := make(map[VerifySubjectType]VerifySubjectRule, len(rules))
	for _, rule := range rules {
		byType[rule.Type] = rule
	}
	return byType
}

func verifyPredicateAllowed(rule VerifySubjectRule, predicate VerifyPredicate) bool {
	for _, allowed := range rule.AllowedPredicates {
		if allowed == predicate {
			return true
		}
	}
	return false
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
