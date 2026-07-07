package check

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spacehz-lab/cal/internal/model"
)

func registerTextPredicates(c *Checker) {
	textSubjects := []model.VerifySubjectType{model.VerifySubjectFile, model.VerifySubjectStdout, model.VerifySubjectStderr}
	scalarSubjects := []model.VerifySubjectType{model.VerifySubjectStdout, model.VerifySubjectStderr, model.VerifySubjectExitCode}
	c.register(predicate{
		name:     model.VerifyPredicateEquals,
		subjects: scalarSubjects,
		params:   []paramRule{{name: paramValue, required: true}},
		run:      checkEquals,
	})
	c.register(predicate{
		name:     model.VerifyPredicateNotEquals,
		subjects: scalarSubjects,
		params:   []paramRule{{name: paramValue, required: true}},
		run:      checkNotEquals,
	})
	c.register(predicate{
		name:     model.VerifyPredicateContains,
		subjects: textSubjects,
		params:   []paramRule{{name: paramValue, required: true}},
		run:      checkContains,
	})
	c.register(predicate{
		name:     model.VerifyPredicateContainsAny,
		subjects: textSubjects,
		params:   []paramRule{{name: paramValues, required: true}},
		run:      checkContainsAny,
	})
	c.register(predicate{
		name:     model.VerifyPredicateRegex,
		subjects: textSubjects,
		params:   []paramRule{{name: paramPattern, required: true}},
		run:      checkRegex,
	})
}

func checkEquals(ctx *predicateContext) error {
	want := ctx.check.Params[paramValue]
	if fmt.Sprint(ctx.subject.value) != fmt.Sprint(want) {
		return fmt.Errorf("verify %s equals failed: got %q want %q", ctx.subject.label, fmt.Sprint(ctx.subject.value), fmt.Sprint(want))
	}
	return nil
}

func checkNotEquals(ctx *predicateContext) error {
	want := ctx.check.Params[paramValue]
	if fmt.Sprint(ctx.subject.value) == fmt.Sprint(want) {
		return fmt.Errorf("verify %s not_equals failed: got %q", ctx.subject.label, fmt.Sprint(ctx.subject.value))
	}
	return nil
}

func checkContains(ctx *predicateContext) error {
	needle := stringParam(ctx.check.Params, paramValue)
	text, err := subjectText(ctx.subject)
	if err != nil {
		return err
	}
	if !strings.Contains(text, needle) {
		return fmt.Errorf("verify %s contains failed", ctx.subject.label)
	}
	return nil
}

func checkContainsAny(ctx *predicateContext) error {
	values := stringListParam(ctx.check.Params, paramValues)
	text, err := subjectText(ctx.subject)
	if err != nil {
		return err
	}
	for _, value := range values {
		if strings.Contains(text, value) {
			return nil
		}
	}
	return fmt.Errorf("verify %s contains_any failed", ctx.subject.label)
}

func checkRegex(ctx *predicateContext) error {
	pattern := stringParam(ctx.check.Params, paramPattern)
	text, err := subjectText(ctx.subject)
	if err != nil {
		return err
	}
	ok, err := regexp.MatchString(pattern, text)
	if err != nil {
		return fmt.Errorf("verify regex pattern: %w", err)
	}
	if !ok {
		return fmt.Errorf("verify %s regex failed", ctx.subject.label)
	}
	return nil
}
