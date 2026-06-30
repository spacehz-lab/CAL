package core

// VerifyLevel describes the confidence level for a binding verification.
type VerifyLevel string

const (
	VerifyLevelL0 VerifyLevel = "L0"
	VerifyLevelL1 VerifyLevel = "L1"
	VerifyLevelL2 VerifyLevel = "L2"
	VerifyLevelL3 VerifyLevel = "L3"
)

// VerifyMethod identifies how verification evidence is obtained.
type VerifyMethod string

const (
	VerifyMethodExecute  VerifyMethod = "execute"
	VerifyMethodContract VerifyMethod = "contract"
)

// VerifyPredicate identifies a built-in deterministic check.
type VerifyPredicate string

const (
	VerifyPredicateEquals              VerifyPredicate = "equals"
	VerifyPredicateNotEquals           VerifyPredicate = "not_equals"
	VerifyPredicateExists              VerifyPredicate = "exists"
	VerifyPredicateNonEmpty            VerifyPredicate = "non_empty"
	VerifyPredicateFormat              VerifyPredicate = "format"
	VerifyPredicateContains            VerifyPredicate = "contains"
	VerifyPredicateContainsAny         VerifyPredicate = "contains_any"
	VerifyPredicateRegex               VerifyPredicate = "regex"
	VerifyPredicateBytesEqualTransform VerifyPredicate = "bytes_equal_transform"
	VerifyPredicateHashLineMatches     VerifyPredicate = "hash_line_matches"
)

// VerifySubjectType identifies where a deterministic check reads evidence.
type VerifySubjectType string

const (
	VerifySubjectFile     VerifySubjectType = "file"
	VerifySubjectStdout   VerifySubjectType = "stdout"
	VerifySubjectStderr   VerifySubjectType = "stderr"
	VerifySubjectExitCode VerifySubjectType = "exit_code"
)

// VerifySpec describes deterministic checks for a binding.
type VerifySpec struct {
	Level  VerifyLevel   `json:"level"`
	Method VerifyMethod  `json:"method"`
	Checks []VerifyCheck `json:"checks,omitempty"`
}

// VerifySubject describes the typed evidence source for one check.
type VerifySubject struct {
	Type  VerifySubjectType `json:"type"`
	Input string            `json:"input,omitempty"`
}

// VerifyCheck describes one built-in deterministic check.
type VerifyCheck struct {
	Subject   VerifySubject   `json:"subject"`
	Predicate VerifyPredicate `json:"predicate"`
	Params    map[string]any  `json:"params,omitempty"`
}

// VerifyParamRule describes one predicate parameter constraint.
type VerifyParamRule struct {
	Required      bool     `json:"required,omitempty"`
	AllowedValues []string `json:"allowed_values,omitempty"`
}

// VerifySubjectRule describes which predicates are valid for one subject type.
type VerifySubjectRule struct {
	Type              VerifySubjectType                              `json:"type"`
	RequiresInput     bool                                           `json:"requires_input,omitempty"`
	AllowedPredicates []VerifyPredicate                              `json:"allowed_predicates"`
	RequiredParams    map[VerifyPredicate][]string                   `json:"required_params,omitempty"`
	ParamRules        map[VerifyPredicate]map[string]VerifyParamRule `json:"param_rules,omitempty"`
}

// VerifySubjectRules returns the code-owned VerifySpec subject contract.
func VerifySubjectRules() []VerifySubjectRule {
	return []VerifySubjectRule{
		{
			Type:          VerifySubjectFile,
			RequiresInput: true,
			AllowedPredicates: []VerifyPredicate{
				VerifyPredicateExists,
				VerifyPredicateNonEmpty,
				VerifyPredicateFormat,
				VerifyPredicateContains,
				VerifyPredicateContainsAny,
				VerifyPredicateRegex,
				VerifyPredicateBytesEqualTransform,
				VerifyPredicateHashLineMatches,
			},
			RequiredParams: verifyRequiredParams(
				VerifyPredicateFormat,
				VerifyPredicateContains,
				VerifyPredicateContainsAny,
				VerifyPredicateRegex,
				VerifyPredicateBytesEqualTransform,
				VerifyPredicateHashLineMatches,
			),
			ParamRules: verifyParamRules(
				VerifyPredicateFormat,
				VerifyPredicateContains,
				VerifyPredicateContainsAny,
				VerifyPredicateRegex,
				VerifyPredicateBytesEqualTransform,
				VerifyPredicateHashLineMatches,
			),
		},
		{
			Type: VerifySubjectStdout,
			AllowedPredicates: []VerifyPredicate{
				VerifyPredicateEquals,
				VerifyPredicateNotEquals,
				VerifyPredicateNonEmpty,
				VerifyPredicateContains,
				VerifyPredicateContainsAny,
				VerifyPredicateRegex,
				VerifyPredicateHashLineMatches,
			},
			RequiredParams: verifyRequiredParams(
				VerifyPredicateEquals,
				VerifyPredicateNotEquals,
				VerifyPredicateContains,
				VerifyPredicateContainsAny,
				VerifyPredicateRegex,
				VerifyPredicateHashLineMatches,
			),
			ParamRules: verifyParamRules(
				VerifyPredicateEquals,
				VerifyPredicateNotEquals,
				VerifyPredicateContains,
				VerifyPredicateContainsAny,
				VerifyPredicateRegex,
				VerifyPredicateHashLineMatches,
			),
		},
		{
			Type: VerifySubjectStderr,
			AllowedPredicates: []VerifyPredicate{
				VerifyPredicateEquals,
				VerifyPredicateNotEquals,
				VerifyPredicateNonEmpty,
				VerifyPredicateContains,
				VerifyPredicateContainsAny,
				VerifyPredicateRegex,
				VerifyPredicateHashLineMatches,
			},
			RequiredParams: verifyRequiredParams(
				VerifyPredicateEquals,
				VerifyPredicateNotEquals,
				VerifyPredicateContains,
				VerifyPredicateContainsAny,
				VerifyPredicateRegex,
				VerifyPredicateHashLineMatches,
			),
			ParamRules: verifyParamRules(
				VerifyPredicateEquals,
				VerifyPredicateNotEquals,
				VerifyPredicateContains,
				VerifyPredicateContainsAny,
				VerifyPredicateRegex,
				VerifyPredicateHashLineMatches,
			),
		},
		{
			Type: VerifySubjectExitCode,
			AllowedPredicates: []VerifyPredicate{
				VerifyPredicateEquals,
				VerifyPredicateNotEquals,
			},
			RequiredParams: verifyRequiredParams(
				VerifyPredicateEquals,
				VerifyPredicateNotEquals,
			),
			ParamRules: verifyParamRules(
				VerifyPredicateEquals,
				VerifyPredicateNotEquals,
			),
		},
	}
}

// VerifyLevelRank returns a comparable rank for verification levels.
func VerifyLevelRank(level VerifyLevel) int {
	switch level {
	case VerifyLevelL1:
		return 1
	case VerifyLevelL2:
		return 2
	case VerifyLevelL3:
		return 3
	default:
		return 0
	}
}

func verifyRequiredParams(predicates ...VerifyPredicate) map[VerifyPredicate][]string {
	params := map[VerifyPredicate][]string{}
	for _, predicate := range predicates {
		rules := verifyPredicateParamRules(predicate)
		for _, name := range verifyPredicateParamOrder(predicate) {
			rule := rules[name]
			if rule.Required {
				params[predicate] = append(params[predicate], name)
			}
		}
	}
	return params
}

func verifyParamRules(predicates ...VerifyPredicate) map[VerifyPredicate]map[string]VerifyParamRule {
	rules := map[VerifyPredicate]map[string]VerifyParamRule{}
	for _, predicate := range predicates {
		if params := verifyPredicateParamRules(predicate); len(params) > 0 {
			rules[predicate] = params
		}
	}
	return rules
}

func verifyPredicateParamRules(predicate VerifyPredicate) map[string]VerifyParamRule {
	switch predicate {
	case VerifyPredicateEquals, VerifyPredicateNotEquals, VerifyPredicateContains:
		return map[string]VerifyParamRule{"value": {Required: true}}
	case VerifyPredicateContainsAny:
		return map[string]VerifyParamRule{"values": {Required: true}}
	case VerifyPredicateRegex:
		return map[string]VerifyParamRule{"pattern": {Required: true}}
	case VerifyPredicateFormat:
		return map[string]VerifyParamRule{"format": {Required: true, AllowedValues: []string{"pdf", "png", "json", "text"}}}
	case VerifyPredicateBytesEqualTransform:
		return map[string]VerifyParamRule{
			"source":    {Required: true},
			"transform": {Required: true, AllowedValues: []string{"base64_encode", "base64_decode"}},
		}
	case VerifyPredicateHashLineMatches:
		return map[string]VerifyParamRule{
			"source":    {Required: true},
			"algorithm": {Required: true, AllowedValues: []string{"sha1", "sha256", "sha-1", "sha-256", "sha_1", "sha_256", "sha 1", "sha 256"}},
		}
	default:
		return nil
	}
}

func verifyPredicateParamOrder(predicate VerifyPredicate) []string {
	switch predicate {
	case VerifyPredicateEquals, VerifyPredicateNotEquals, VerifyPredicateContains:
		return []string{"value"}
	case VerifyPredicateContainsAny:
		return []string{"values"}
	case VerifyPredicateRegex:
		return []string{"pattern"}
	case VerifyPredicateFormat:
		return []string{"format"}
	case VerifyPredicateBytesEqualTransform:
		return []string{"source", "transform"}
	case VerifyPredicateHashLineMatches:
		return []string{"source", "algorithm"}
	default:
		return nil
	}
}
