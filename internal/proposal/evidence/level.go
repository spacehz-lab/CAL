package evidence

import "github.com/spacehz-lab/cal/internal/model"

type evidenceStrength int

const (
	evidenceStrengthNone evidenceStrength = iota
	evidenceStrengthProcess
	evidenceStrengthArtifact
	evidenceStrengthSemantic
)

func deriveLevel(verify model.VerifySpec) model.VerifyLevel {
	switch verify.Method {
	case model.VerifyMethodContract:
		return model.VerifyLevelL1
	case model.VerifyMethodExecute:
	default:
		return verify.Level
	}
	if len(verify.Checks) == 0 {
		return model.VerifyLevelL0
	}
	strength := evidenceStrengthProcess
	for _, check := range verify.Checks {
		if current := checkStrength(check); current > strength {
			strength = current
		}
	}
	switch strength {
	case evidenceStrengthSemantic:
		return model.VerifyLevelL3
	case evidenceStrengthArtifact:
		return model.VerifyLevelL2
	case evidenceStrengthProcess:
		return model.VerifyLevelL1
	default:
		return model.VerifyLevelL0
	}
}

func checkStrength(check model.VerifyCheck) evidenceStrength {
	switch check.Predicate {
	case model.VerifyPredicateContains,
		model.VerifyPredicateContainsAny,
		model.VerifyPredicateBytesEqualTransform,
		model.VerifyPredicateHashLineMatches,
		model.VerifyPredicateArchiveContainsInput,
		model.VerifyPredicateJSONQueryMatches,
		model.VerifyPredicateJSONEquivalent,
		model.VerifyPredicateTextTransformMatches,
		model.VerifyPredicateLineCountMatches,
		model.VerifyPredicateTextFilterMatches,
		model.VerifyPredicateDelimitedColumnMatch:
		return evidenceStrengthSemantic
	case model.VerifyPredicateNonEmpty,
		model.VerifyPredicateFormat,
		model.VerifyPredicateRegex:
		return evidenceStrengthArtifact
	case model.VerifyPredicateExists,
		model.VerifyPredicateEquals,
		model.VerifyPredicateNotEquals:
		return evidenceStrengthProcess
	default:
		return evidenceStrengthNone
	}
}
