package check

import (
	"fmt"

	"github.com/spacehz-lab/cal/internal/model"
)

type predicate struct {
	name     model.VerifyPredicate
	subjects []model.VerifySubjectType
	params   []paramRule
	run      predicateRunner
}

type paramRule struct {
	name          string
	required      bool
	allowedValues []string
}

type predicateContext struct {
	check   *model.VerifyCheck
	subject *checkSubject
}

type predicateRunner func(*predicateContext) error

func (c *Checker) register(predicate predicate) {
	c.predicates[predicate.name] = predicate
}

func (c *Checker) runPredicate(check *model.VerifyCheck, subject *checkSubject) error {
	predicate, ok := c.predicates[check.Predicate]
	if !ok {
		return fmt.Errorf("verify predicate %q is not supported", check.Predicate)
	}
	return predicate.run(&predicateContext{check: check, subject: subject})
}

func (p predicate) allowsSubject(subject model.VerifySubjectType) bool {
	for _, allowed := range p.subjects {
		if allowed == subject {
			return true
		}
	}
	return false
}

var subjectOrder = []model.VerifySubjectType{
	model.VerifySubjectFile,
	model.VerifySubjectStdout,
	model.VerifySubjectStderr,
	model.VerifySubjectExitCode,
}

var predicateOrder = []model.VerifyPredicate{
	model.VerifyPredicateEquals,
	model.VerifyPredicateNotEquals,
	model.VerifyPredicateExists,
	model.VerifyPredicateNonEmpty,
	model.VerifyPredicateFormat,
	model.VerifyPredicateContains,
	model.VerifyPredicateContainsAny,
	model.VerifyPredicateRegex,
	model.VerifyPredicateBytesEqualTransform,
	model.VerifyPredicateHashLineMatches,
	model.VerifyPredicateArchiveContainsInput,
	model.VerifyPredicateJSONQueryMatches,
}

func subjectRequiresInput(subject model.VerifySubjectType) (bool, bool) {
	switch subject {
	case model.VerifySubjectFile:
		return true, true
	case model.VerifySubjectStdout, model.VerifySubjectStderr, model.VerifySubjectExitCode:
		return false, true
	default:
		return false, false
	}
}
