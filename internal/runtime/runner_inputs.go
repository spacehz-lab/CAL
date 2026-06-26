package runtime

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"

	"github.com/spacehz-lab/cal/internal/core"
)

var inputPlaceholderPattern = regexp.MustCompile(`\{\{([A-Za-z0-9_]+)\}\}`)

// ValidateBindingInputs checks inputs against the selected binding contract.
func (runner Runner) Validate(binding core.Binding, inputs map[string]any) error {
	required, err := runner.RequiredInputs(binding.Execution)
	if err != nil {
		return err
	}
	for _, name := range required {
		value, ok := inputs[name]
		if !ok || value == nil || value == "" {
			return fmt.Errorf("missing required input: %s", name)
		}
	}
	for name, constraint := range binding.InputConstraints {
		value, ok := inputs[name]
		if !ok {
			continue
		}
		if err := validateInputConstraint(name, value, constraint); err != nil {
			return err
		}
	}
	return nil
}

// RequiredInputs returns sorted runtime input names referenced by an execution.
func (runner Runner) RequiredInputs(execution core.Execution) ([]string, error) {
	required := map[string]struct{}{}
	if execution.Kind == core.ExecutionKindCLI {
		args, err := executionArgs(execution)
		if err != nil {
			return nil, err
		}
		for _, arg := range args {
			for _, match := range inputPlaceholderPattern.FindAllStringSubmatch(arg, -1) {
				required[match[1]] = struct{}{}
			}
		}
		if input, ok, err := stdoutPathInput(execution); err != nil {
			return nil, err
		} else if ok {
			required[input] = struct{}{}
		}
	}
	names := make([]string, 0, len(required))
	for name := range required {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

func validateInputConstraint(name string, value any, constraint any) error {
	fields, ok := constraint.(map[string]any)
	if !ok {
		return fmt.Errorf("input constraint %q must be an object", name)
	}
	enumValue, ok := fields["enum"]
	if !ok {
		return nil
	}
	values, err := enumValues(enumValue)
	if err != nil {
		return fmt.Errorf("input constraint %q enum: %w", name, err)
	}
	for _, allowed := range values {
		if reflect.DeepEqual(value, allowed) || fmt.Sprint(value) == fmt.Sprint(allowed) {
			return nil
		}
	}
	return fmt.Errorf("input %q value %q is not allowed by selected binding", name, fmt.Sprint(value))
}

func enumValues(value any) ([]any, error) {
	switch typed := value.(type) {
	case []any:
		return typed, nil
	case []string:
		values := make([]any, len(typed))
		for index, item := range typed {
			values[index] = item
		}
		return values, nil
	default:
		return nil, fmt.Errorf("must be an array")
	}
}

func executionArgs(execution core.Execution) ([]string, error) {
	value, ok := execution.Spec[core.ExecutionSpecArgs]
	if !ok {
		return nil, fmt.Errorf("cli execution args are required")
	}
	return stringSlice(value)
}
