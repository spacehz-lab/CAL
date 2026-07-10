package check

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/spacehz-lab/cal/internal/model"
)

func registerJSONPredicates(c *Checker) {
	c.register(predicate{
		name:     model.VerifyPredicateJSONQueryMatches,
		subjects: []model.VerifySubjectType{model.VerifySubjectFile, model.VerifySubjectStdout, model.VerifySubjectStderr},
		params: []paramRule{
			{name: paramSource, required: true},
			{name: paramQuery, required: true},
		},
		run: checkJSONQueryMatches,
	})
	c.register(predicate{
		name:     model.VerifyPredicateJSONEquivalent,
		subjects: []model.VerifySubjectType{model.VerifySubjectFile, model.VerifySubjectStdout, model.VerifySubjectStderr},
		params:   []paramRule{{name: paramSource, required: true}},
		run:      checkJSONEquivalent,
	})
	c.register(predicate{
		name:     model.VerifyPredicateJSONFieldEquals,
		subjects: []model.VerifySubjectType{model.VerifySubjectFile, model.VerifySubjectStdout, model.VerifySubjectStderr},
		params: []paramRule{
			{name: paramQuery, required: true},
			{name: paramValue, required: true},
		},
		run: checkJSONFieldEquals,
	})
	c.register(predicate{
		name:     model.VerifyPredicateJSONFieldMatchesSource,
		subjects: []model.VerifySubjectType{model.VerifySubjectFile, model.VerifySubjectStdout, model.VerifySubjectStderr},
		params: []paramRule{
			{name: paramQuery, required: true},
			{name: paramSource, required: true},
			{name: paramProperty, required: true, allowedValues: []string{sourcePropertyBasename, sourcePropertyBytes, sourcePropertySHA256}},
		},
		run: checkJSONFieldMatchesSource,
	})
}

func checkJSONQueryMatches(ctx *predicateContext) error {
	source := pathParam(ctx.check.Params, paramSource, ctx.subject.inputs)
	query := valueParam(ctx.check.Params, paramQuery, ctx.subject.inputs)
	sourceBytes, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}
	want, err := queryJSON(sourceBytes, query)
	if err != nil {
		return err
	}
	text, err := subjectText(ctx.subject)
	if err != nil {
		return err
	}
	if strings.TrimSpace(text) != strings.TrimSpace(want) {
		return fmt.Errorf("verify json_query_matches failed")
	}
	return nil
}

func checkJSONEquivalent(ctx *predicateContext) error {
	source := pathParam(ctx.check.Params, paramSource, ctx.subject.inputs)
	sourceBytes, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}
	text, err := subjectText(ctx.subject)
	if err != nil {
		return err
	}
	sourceValue, err := parseJSONValue(sourceBytes)
	if err != nil {
		return fmt.Errorf("parse json source: %w", err)
	}
	targetValue, err := parseJSONValue([]byte(text))
	if err != nil {
		return fmt.Errorf("parse json target: %w", err)
	}
	if !reflect.DeepEqual(sourceValue, targetValue) {
		return fmt.Errorf("verify json_equivalent failed")
	}
	return nil
}

func checkJSONFieldEquals(ctx *predicateContext) error {
	got, err := jsonFieldText(ctx)
	if err != nil {
		return err
	}
	want := valueParam(ctx.check.Params, paramValue, ctx.subject.inputs)
	if strings.TrimSpace(got) != strings.TrimSpace(want) {
		return fmt.Errorf("verify json_field_equals failed")
	}
	return nil
}

func checkJSONFieldMatchesSource(ctx *predicateContext) error {
	got, err := jsonFieldText(ctx)
	if err != nil {
		return err
	}
	want, err := sourcePropertyValue(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(got) != want {
		return fmt.Errorf("verify json_field_matches_source failed")
	}
	return nil
}

func jsonFieldText(ctx *predicateContext) (string, error) {
	text, err := subjectText(ctx.subject)
	if err != nil {
		return "", err
	}
	value, err := parseJSONValue([]byte(text))
	if err != nil {
		return "", fmt.Errorf("parse json target: %w", err)
	}
	current, err := walkJSON(value, valueParam(ctx.check.Params, paramQuery, ctx.subject.inputs))
	if err != nil {
		return "", err
	}
	return jsonScalarText(current)
}

func sourcePropertyValue(ctx *predicateContext) (string, error) {
	source := pathParam(ctx.check.Params, paramSource, ctx.subject.inputs)
	content, err := os.ReadFile(source)
	if err != nil {
		return "", fmt.Errorf("read source: %w", err)
	}
	switch strings.ToLower(stringParam(ctx.check.Params, paramProperty)) {
	case sourcePropertyBasename:
		return filepath.Base(source), nil
	case sourcePropertyBytes:
		return strconv.Itoa(len(content)), nil
	case sourcePropertySHA256:
		sum := sha256.Sum256(content)
		return fmt.Sprintf("%x", sum), nil
	default:
		return "", fmt.Errorf("source property %q is not supported", stringParam(ctx.check.Params, paramProperty))
	}
}

func queryJSON(content []byte, query string) (string, error) {
	value, err := parseJSONValue(content)
	if err != nil {
		return "", fmt.Errorf("parse json source: %w", err)
	}
	current, err := walkJSON(value, query)
	if err != nil {
		return "", err
	}
	return jsonScalarText(current)
}

func parseJSONValue(content []byte) (any, error) {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

func walkJSON(value any, query string) (any, error) {
	path := strings.TrimSpace(query)
	path = strings.TrimPrefix(path, "$")
	path = strings.TrimPrefix(path, ".")
	if path == "" {
		return value, nil
	}
	current := value
	for _, part := range strings.Split(path, ".") {
		if part == "" {
			return nil, fmt.Errorf("json query %q is invalid", query)
		}
		switch typed := current.(type) {
		case map[string]any:
			next, ok := typed[part]
			if !ok {
				return nil, fmt.Errorf("json query %q missing key %q", query, part)
			}
			current = next
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(typed) {
				return nil, fmt.Errorf("json query %q invalid index %q", query, part)
			}
			current = typed[index]
		default:
			return nil, fmt.Errorf("json query %q cannot descend into %T", query, current)
		}
	}
	return current, nil
}

func jsonScalarText(value any) (string, error) {
	switch typed := value.(type) {
	case nil:
		return "null", nil
	case string:
		return typed, nil
	case json.Number:
		return typed.String(), nil
	case bool:
		if typed {
			return "true", nil
		}
		return "false", nil
	default:
		content, err := json.Marshal(typed)
		if err != nil {
			return "", fmt.Errorf("format json query result: %w", err)
		}
		return string(content), nil
	}
}

func valueParam(params map[string]any, key string, inputs map[string]any) string {
	value := stringParam(params, key)
	if inputValue, ok := inputs[value]; ok && strings.TrimSpace(fmt.Sprint(inputValue)) != "" {
		return strings.TrimSpace(fmt.Sprint(inputValue))
	}
	return value
}
