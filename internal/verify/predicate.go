package verify

import (
	"fmt"

	"github.com/spacehz-lab/cal/internal/core"
)

type predicateHandler func(core.VerifyCheck, checkSubject) error

var predicateHandlers = map[core.VerifyPredicate]predicateHandler{
	core.VerifyPredicateEquals:              checkEquals,
	core.VerifyPredicateNotEquals:           checkNotEquals,
	core.VerifyPredicateExists:              checkExists,
	core.VerifyPredicateNonEmpty:            checkNonEmpty,
	core.VerifyPredicateFormat:              checkFormatPredicate,
	core.VerifyPredicateContains:            checkContains,
	core.VerifyPredicateContainsAny:         checkContainsAny,
	core.VerifyPredicateRegex:               checkRegex,
	core.VerifyPredicateBytesEqualTransform: checkBytesEqualTransform,
	core.VerifyPredicateHashLineMatches:     checkHashLineMatches,
}

func evaluatePredicate(check core.VerifyCheck, subject checkSubject) error {
	handler, ok := predicateHandlers[check.Predicate]
	if !ok {
		return fmt.Errorf("verify predicate %q is not supported", check.Predicate)
	}
	return handler(check, subject)
}

func hasPredicateHandler(predicate core.VerifyPredicate) bool {
	_, ok := predicateHandlers[predicate]
	return ok
}
