package evidence

import (
	"strings"
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

func TestValidatePredicateParamsAcceptsJSONFieldMatchesSource(t *testing.T) {
	check := model.VerifyCheck{
		Predicate: model.VerifyPredicateJSONFieldMatchesSource,
		Params: map[string]any{
			paramQuery:    ".sha256",
			paramSource:   "source",
			paramProperty: sourcePropertySHA256,
		},
	}

	reason := validatePredicateParams(check, predicateRuleMap(verifyPredicateRules()))

	if reason != "" {
		t.Fatalf("reason = %q, want keep", reason)
	}
}

func TestValidatePredicateParamsRejectsUnsupportedJSONSourceProperty(t *testing.T) {
	check := model.VerifyCheck{
		Predicate: model.VerifyPredicateJSONFieldMatchesSource,
		Params: map[string]any{
			paramQuery:    ".mode",
			paramSource:   "source",
			paramProperty: "mode",
		},
	}

	reason := validatePredicateParams(check, predicateRuleMap(verifyPredicateRules()))

	if reason != "invalid predicate param" {
		t.Fatalf("reason = %q, want invalid predicate param", reason)
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

func TestLineCountMatchesRuleDescribesNumericReport(t *testing.T) {
	rule := predicateRuleMap(verifyPredicateRules())[model.VerifyPredicateLineCountMatches]

	if rule.Description == "" {
		t.Fatal("line_count_matches description is empty")
	}
	for _, want := range []string{"numeric line-count report", "first integer", "number of lines in source"} {
		if !strings.Contains(rule.Description, want) {
			t.Fatalf("line_count_matches description = %q, want %q", rule.Description, want)
		}
	}
}
