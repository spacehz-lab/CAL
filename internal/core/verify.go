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

// VerifySubjectRule describes which predicates are valid for one subject type.
type VerifySubjectRule struct {
	Type              VerifySubjectType            `json:"type"`
	RequiresInput     bool                         `json:"requires_input,omitempty"`
	AllowedPredicates []VerifyPredicate            `json:"allowed_predicates"`
	RequiredParams    map[VerifyPredicate][]string `json:"required_params,omitempty"`
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
		switch predicate {
		case VerifyPredicateEquals, VerifyPredicateNotEquals, VerifyPredicateContains:
			params[predicate] = []string{"value"}
		case VerifyPredicateContainsAny:
			params[predicate] = []string{"values"}
		case VerifyPredicateRegex:
			params[predicate] = []string{"pattern"}
		case VerifyPredicateFormat:
			params[predicate] = []string{"format"}
		case VerifyPredicateBytesEqualTransform:
			params[predicate] = []string{"source", "transform"}
		case VerifyPredicateHashLineMatches:
			params[predicate] = []string{"source", "algorithm"}
		}
	}
	return params
}
