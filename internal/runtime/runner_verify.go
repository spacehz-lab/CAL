package runtime

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/spacehz-lab/cal/internal/core"
)

func evaluateVerifySpec(ctx context.Context, verify core.VerifySpec, inputs map[string]any, result ExecutionResult) ([]core.EvidenceRef, map[string]any, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if err := core.ValidateVerifySpec(verify); err != nil {
		return nil, nil, err
	}
	if verify.Method != core.VerifyMethodExecute {
		return nil, nil, fmt.Errorf("verify method %s is not executable", verify.Method)
	}
	if verify.Level == core.VerifyLevelL0 {
		return nil, nil, fmt.Errorf("verify level L0 is not executable")
	}
	evidence := make([]core.EvidenceRef, 0, len(verify.Checks))
	outputs := map[string]any{}
	for index, check := range verify.Checks {
		item, output, err := evaluateCheck(check, inputs, result, index)
		if err != nil {
			return nil, nil, err
		}
		evidence = append(evidence, item)
		for key, value := range output {
			outputs[key] = value
		}
	}
	return evidence, outputs, nil
}

func evaluateCheck(check core.VerifyCheck, inputs map[string]any, result ExecutionResult, index int) (core.EvidenceRef, map[string]any, error) {
	subject, err := subjectValue(check.Subject, inputs, result)
	if err != nil {
		return core.EvidenceRef{}, nil, err
	}
	if err := checkPredicate(check, subject); err != nil {
		return core.EvidenceRef{}, nil, err
	}
	id := fmt.Sprintf("check_%d_%s_%s", index+1, subject.label, check.Predicate)
	return core.EvidenceRef{
		ID:   id,
		Type: string(check.Predicate),
		Content: map[string]any{
			"subject":   check.Subject,
			"predicate": check.Predicate,
		},
	}, subject.outputs, nil
}

type checkSubject struct {
	value   any
	path    string
	label   string
	inputs  map[string]any
	outputs map[string]any
}

func subjectValue(subject core.VerifySubject, inputs map[string]any, result ExecutionResult) (checkSubject, error) {
	switch subject.Type {
	case core.VerifySubjectFile:
		return pathSubject(subject.Input, inputs)
	case core.VerifySubjectStdout:
		return scalarSubject(string(core.VerifySubjectStdout), result.Stdout, inputs)
	case core.VerifySubjectStderr:
		return scalarSubject(string(core.VerifySubjectStderr), result.Stderr, inputs)
	case core.VerifySubjectExitCode:
		return scalarSubject(string(core.VerifySubjectExitCode), result.ExitCode, inputs)
	default:
		return checkSubject{}, fmt.Errorf("verify subject type %q is not supported", subject.Type)
	}
}

func pathSubject(input string, inputs map[string]any) (checkSubject, error) {
	path, ok := inputs[input].(string)
	if !ok || strings.TrimSpace(path) == "" {
		return checkSubject{}, fmt.Errorf("verify subject %q path input is required", input)
	}
	return checkSubject{path: path, value: path, label: input, inputs: inputs, outputs: map[string]any{input: path}}, nil
}

func scalarSubject(label string, value any, inputs map[string]any) (checkSubject, error) {
	return checkSubject{value: value, label: label, inputs: inputs, outputs: map[string]any{label: value}}, nil
}

func checkPredicate(check core.VerifyCheck, subject checkSubject) error {
	switch check.Predicate {
	case core.VerifyPredicateEquals:
		want, ok := check.Params["value"]
		if !ok {
			return fmt.Errorf("verify equals requires params.value")
		}
		if fmt.Sprint(subject.value) != fmt.Sprint(want) {
			return fmt.Errorf("verify %s equals failed: got %q want %q", subject.label, fmt.Sprint(subject.value), fmt.Sprint(want))
		}
	case core.VerifyPredicateNotEquals:
		want, ok := check.Params["value"]
		if !ok {
			return fmt.Errorf("verify not_equals requires params.value")
		}
		if fmt.Sprint(subject.value) == fmt.Sprint(want) {
			return fmt.Errorf("verify %s not_equals failed: got %q", subject.label, fmt.Sprint(subject.value))
		}
	case core.VerifyPredicateExists:
		if _, err := os.Stat(subject.path); err != nil {
			return fmt.Errorf("verify %s exists failed: %w", subject.label, err)
		}
	case core.VerifyPredicateNonEmpty:
		if subject.path != "" {
			info, err := os.Stat(subject.path)
			if err != nil {
				return fmt.Errorf("verify %s non_empty failed: %w", subject.label, err)
			}
			if info.Size() == 0 {
				return fmt.Errorf("verify %s non_empty failed: empty artifact", subject.label)
			}
			return nil
		}
		if strings.TrimSpace(fmt.Sprint(subject.value)) == "" {
			return fmt.Errorf("verify %s non_empty failed", subject.label)
		}
	case core.VerifyPredicateContains:
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
	case core.VerifyPredicateContainsAny:
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
	case core.VerifyPredicateRegex:
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
	case core.VerifyPredicateFormat:
		format := stringParam(check.Params, "format")
		if format == "" {
			return fmt.Errorf("verify format requires params.format")
		}
		if err := checkFormat(subject.path, format); err != nil {
			return err
		}
	case core.VerifyPredicateBytesEqualTransform:
		if err := checkBytesEqualTransform(subject.path, check.Params, subject.inputs); err != nil {
			return err
		}
	case core.VerifyPredicateHashLineMatches:
		if err := checkHashLineMatches(subject, check.Params, subject.inputs); err != nil {
			return err
		}
	default:
		return fmt.Errorf("verify predicate %q is not supported", check.Predicate)
	}
	return nil
}

func subjectText(subject checkSubject) (string, error) {
	if subject.path == "" {
		return fmt.Sprint(subject.value), nil
	}
	content, err := os.ReadFile(subject.path)
	if err != nil {
		return "", fmt.Errorf("verify subject read: %w", err)
	}
	return string(content), nil
}

func checkFormat(path, format string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("verify format read: %w", err)
	}
	switch strings.ToLower(format) {
	case "pdf":
		if !bytes.HasPrefix(content, []byte("%PDF")) {
			return fmt.Errorf("verify format pdf failed")
		}
	case "png":
		if !bytes.HasPrefix(content, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
			return fmt.Errorf("verify format png failed")
		}
	case "json":
		var value any
		if err := json.Unmarshal(content, &value); err != nil {
			return fmt.Errorf("verify format json failed: %w", err)
		}
	case "text":
		if !utf8.Valid(content) {
			return fmt.Errorf("verify format text failed")
		}
	default:
		return fmt.Errorf("verify format %q is not supported", format)
	}
	return nil
}

func checkBytesEqualTransform(target string, params map[string]any, inputs map[string]any) error {
	source := pathParam(params, "source", inputs)
	transform := stringParam(params, "transform")
	if source == "" || transform == "" {
		return fmt.Errorf("verify bytes_equal_transform requires source and transform")
	}
	sourceBytes, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}
	targetBytes, err := os.ReadFile(target)
	if err != nil {
		return fmt.Errorf("read target: %w", err)
	}
	expected, err := transformBytes(sourceBytes, transform)
	if err != nil {
		return err
	}
	if !bytes.Equal(bytes.TrimSpace(targetBytes), bytes.TrimSpace(expected)) {
		return fmt.Errorf("verify bytes_equal_transform failed")
	}
	return nil
}

func transformBytes(content []byte, transform string) ([]byte, error) {
	switch strings.ToLower(transform) {
	case "base64_encode":
		out := make([]byte, base64.StdEncoding.EncodedLen(len(content)))
		base64.StdEncoding.Encode(out, content)
		return out, nil
	case "base64_decode":
		out := make([]byte, base64.StdEncoding.DecodedLen(len(content)))
		n, err := base64.StdEncoding.Decode(out, bytes.TrimSpace(content))
		if err != nil {
			return nil, err
		}
		return out[:n], nil
	default:
		return nil, fmt.Errorf("verify transform %q is not supported", transform)
	}
}

func checkHashLineMatches(subject checkSubject, params map[string]any, inputs map[string]any) error {
	source := pathParam(params, "source", inputs)
	algorithm := strings.ToLower(stringParam(params, "algorithm"))
	if source == "" || algorithm == "" {
		return fmt.Errorf("verify hash_line_matches requires source and algorithm")
	}
	sourceBytes, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}
	want, err := hashBytes(sourceBytes, algorithm)
	if err != nil {
		return err
	}
	text, err := subjectText(subject)
	if err != nil {
		return err
	}
	if !strings.Contains(strings.ToLower(text), want) {
		return fmt.Errorf("verify hash_line_matches failed")
	}
	return nil
}

func hashBytes(content []byte, algorithm string) (string, error) {
	switch algorithm {
	case "sha1":
		sum := sha1.Sum(content)
		return hex.EncodeToString(sum[:]), nil
	case "sha256":
		sum := sha256.Sum256(content)
		return hex.EncodeToString(sum[:]), nil
	default:
		return "", fmt.Errorf("verify hash algorithm %q is not supported", algorithm)
	}
}

func stringParam(params map[string]any, key string) string {
	if len(params) == 0 {
		return ""
	}
	value, ok := params[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func pathParam(params map[string]any, key string, inputs map[string]any) string {
	value := stringParam(params, key)
	if inputPath, ok := inputs[value].(string); ok && strings.TrimSpace(inputPath) != "" {
		return strings.TrimSpace(inputPath)
	}
	return value
}

func stringListParam(params map[string]any, key string) []string {
	value, ok := params[key]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			text := strings.TrimSpace(fmt.Sprint(item))
			if text != "" {
				values = append(values, text)
			}
		}
		return values
	default:
		return nil
	}
}
