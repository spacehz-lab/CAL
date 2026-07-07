package evidence

import (
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestValidatePredicateParamsRequiresFormat(t *testing.T) {
	check := model.VerifyCheck{Predicate: model.VerifyPredicateFormat, Params: map[string]any{}}

	reason := validatePredicateParams(check, predicateRuleMap(verifyPredicateRules()))

	if reason != "missing predicate param" {
		t.Fatalf("reason = %q, want missing predicate param", reason)
	}
}

func TestValidatePredicateParamsAcceptsAllowedFormat(t *testing.T) {
	check := model.VerifyCheck{Predicate: model.VerifyPredicateFormat, Params: map[string]any{paramFormat: formatJSON}}

	reason := validatePredicateParams(check, predicateRuleMap(verifyPredicateRules()))

	if reason != "" {
		t.Fatalf("reason = %q, want keep", reason)
	}
}

func TestValidatePredicateParamsRejectsUnknownParam(t *testing.T) {
	check := model.VerifyCheck{Predicate: model.VerifyPredicateExists, Params: map[string]any{"extra": "value"}}

	reason := validatePredicateParams(check, predicateRuleMap(verifyPredicateRules()))

	if reason != "unknown predicate param" {
		t.Fatalf("reason = %q, want unknown predicate param", reason)
	}
}

func TestValidatePredicateParamsRejectsUnknownPredicate(t *testing.T) {
	check := model.VerifyCheck{Predicate: "custom"}

	reason := validatePredicateParams(check, predicateRuleMap(verifyPredicateRules()))

	if reason != "unknown predicate" {
		t.Fatalf("reason = %q, want unknown predicate", reason)
	}
}
