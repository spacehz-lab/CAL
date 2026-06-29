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

// VerifySpec describes deterministic checks for a binding.
type VerifySpec struct {
	Level  VerifyLevel   `json:"level"`
	Method VerifyMethod  `json:"method"`
	Checks []VerifyCheck `json:"checks,omitempty"`
}

// VerifyCheck describes one built-in deterministic check.
type VerifyCheck struct {
	Subject   string          `json:"subject"`
	Predicate VerifyPredicate `json:"predicate"`
	Params    map[string]any  `json:"params,omitempty"`
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
