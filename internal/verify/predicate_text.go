package verify

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spacehz-lab/cal/internal/core"
)

func checkEquals(check core.VerifyCheck, subject checkSubject) error {
	want, ok := check.Params["value"]
	if !ok {
		return fmt.Errorf("verify equals requires params.value")
	}
	if fmt.Sprint(subject.value) != fmt.Sprint(want) {
		return fmt.Errorf("verify %s equals failed: got %q want %q", subject.label, fmt.Sprint(subject.value), fmt.Sprint(want))
	}
	return nil
}

func checkNotEquals(check core.VerifyCheck, subject checkSubject) error {
	want, ok := check.Params["value"]
	if !ok {
		return fmt.Errorf("verify not_equals requires params.value")
	}
	if fmt.Sprint(subject.value) == fmt.Sprint(want) {
		return fmt.Errorf("verify %s not_equals failed: got %q", subject.label, fmt.Sprint(subject.value))
	}
	return nil
}

func checkContains(check core.VerifyCheck, subject checkSubject) error {
	needle := stringParam(check.Params, "value")
	if needle == "" {
		return fmt.Errorf("verify contains requires params.value")
	}
	text, err := subjectText(subject)
	if err != nil {
		return err
	}
	if !strings.Contains(text, needle) {
		return fmt.Errorf("verify %s contains failed", subject.label)
	}
	return nil
}

func checkContainsAny(check core.VerifyCheck, subject checkSubject) error {
	values := stringListParam(check.Params, "values")
	if len(values) == 0 {
		return fmt.Errorf("verify contains_any requires params.values")
	}
	text, err := subjectText(subject)
	if err != nil {
		return err
	}
	for _, value := range values {
		if strings.Contains(text, value) {
			return nil
		}
	}
	return fmt.Errorf("verify %s contains_any failed", subject.label)
}

func checkRegex(check core.VerifyCheck, subject checkSubject) error {
	pattern := stringParam(check.Params, "pattern")
	if pattern == "" {
		return fmt.Errorf("verify regex requires params.pattern")
	}
	text, err := subjectText(subject)
	if err != nil {
		return err
	}
	ok, err := regexp.MatchString(pattern, text)
	if err != nil {
		return fmt.Errorf("verify regex pattern: %w", err)
	}
	if !ok {
		return fmt.Errorf("verify %s regex failed", subject.label)
	}
	return nil
}
