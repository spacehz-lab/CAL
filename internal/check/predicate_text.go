package check

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
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
	c.register(predicate{
		name:     model.VerifyPredicateTextTransformMatches,
		subjects: []model.VerifySubjectType{model.VerifySubjectFile},
		params: []paramRule{
			{name: paramSource, required: true},
			{name: paramTransform, required: true, allowedValues: []string{transformUppercase, transformLowercase}},
		},
		run: checkTextTransformMatches,
	})
	c.register(predicate{
		name:     model.VerifyPredicateLineCountMatches,
		subjects: textSubjects,
		params:   []paramRule{{name: paramSource, required: true}},
		run:      checkLineCountMatches,
	})
	c.register(predicate{
		name:     model.VerifyPredicateTextFilterMatches,
		subjects: textSubjects,
		params: []paramRule{
			{name: paramSource, required: true},
			{name: paramPattern, required: true},
		},
		run: checkTextFilterMatches,
	})
	c.register(predicate{
		name:     model.VerifyPredicateDelimitedColumnMatch,
		subjects: textSubjects,
		params: []paramRule{
			{name: paramSource, required: true},
			{name: paramDelimiter, required: true},
			{name: paramColumn, required: true},
		},
		run: checkDelimitedColumnMatches,
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

func checkTextTransformMatches(ctx *predicateContext) error {
	source, err := sourceText(ctx)
	if err != nil {
		return err
	}
	want, err := transformText(source, stringParam(ctx.check.Params, paramTransform))
	if err != nil {
		return err
	}
	got, err := subjectText(ctx.subject)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("verify text_transform_matches failed")
	}
	return nil
}

func checkLineCountMatches(ctx *predicateContext) error {
	source, err := sourceText(ctx)
	if err != nil {
		return err
	}
	want := strings.Count(source, "\n")
	gotText, err := subjectText(ctx.subject)
	if err != nil {
		return err
	}
	got, err := firstInt(gotText)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("verify line_count_matches failed: got %d want %d", got, want)
	}
	return nil
}

func checkTextFilterMatches(ctx *predicateContext) error {
	source, err := sourceText(ctx)
	if err != nil {
		return err
	}
	pattern := valueParam(ctx.check.Params, paramPattern, ctx.subject.inputs)
	want := filterLines(source, pattern)
	got, err := subjectText(ctx.subject)
	if err != nil {
		return err
	}
	if !sameLines(got, want) {
		return fmt.Errorf("verify text_filter_matches failed")
	}
	return nil
}

func checkDelimitedColumnMatches(ctx *predicateContext) error {
	source, err := sourceText(ctx)
	if err != nil {
		return err
	}
	delimiter := valueParam(ctx.check.Params, paramDelimiter, ctx.subject.inputs)
	column, err := strconv.Atoi(valueParam(ctx.check.Params, paramColumn, ctx.subject.inputs))
	if err != nil || column < 1 {
		return fmt.Errorf("verify delimited column is invalid")
	}
	want := delimitedColumn(source, delimiter, column)
	got, err := subjectText(ctx.subject)
	if err != nil {
		return err
	}
	if !sameLines(got, want) {
		return fmt.Errorf("verify delimited_column_matches failed")
	}
	return nil
}

func sourceText(ctx *predicateContext) (string, error) {
	source := pathParam(ctx.check.Params, paramSource, ctx.subject.inputs)
	content, err := readTextFile(source)
	if err != nil {
		return "", fmt.Errorf("read source: %w", err)
	}
	return content, nil
}

func readTextFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func transformText(content string, transform string) (string, error) {
	switch strings.ToLower(transform) {
	case transformUppercase:
		return strings.ToUpper(content), nil
	case transformLowercase:
		return strings.ToLower(content), nil
	default:
		return "", fmt.Errorf("verify text transform %q is not supported", transform)
	}
}

func firstInt(text string) (int, error) {
	match := regexp.MustCompile(`\d+`).FindString(text)
	if match == "" {
		return 0, fmt.Errorf("verify output does not contain an integer")
	}
	return strconv.Atoi(match)
}

func filterLines(content string, pattern string) string {
	var lines []string
	for _, line := range splitLines(content) {
		if strings.Contains(line, pattern) {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

func delimitedColumn(content string, delimiter string, column int) string {
	var values []string
	for _, line := range splitLines(content) {
		parts := strings.Split(line, delimiter)
		if column <= len(parts) {
			values = append(values, parts[column-1])
		}
	}
	return strings.Join(values, "\n")
}

func splitLines(content string) []string {
	trimmed := strings.TrimRight(content, "\r\n")
	if trimmed == "" {
		return nil
	}
	return strings.Split(strings.ReplaceAll(trimmed, "\r\n", "\n"), "\n")
}

func sameLines(got string, want string) bool {
	return strings.Join(splitLines(got), "\n") == strings.Join(splitLines(want), "\n")
}
