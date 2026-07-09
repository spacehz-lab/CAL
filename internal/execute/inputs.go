package execute

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/spacehz-lab/cal/internal/model"
)

var inputPlaceholderPattern = regexp.MustCompile(`\{\{([A-Za-z0-9_]+)\}\}`)

// RequiredInputs returns sorted runtime input names referenced by an execution.
func RequiredInputs(execution *model.Execution) ([]string, error) {
	if execution == nil {
		return nil, ErrNilExecution
	}
	required := map[string]struct{}{}
	switch execution.Kind {
	case model.ExecutionKindCLI:
		args, err := executionArgs(execution)
		if err != nil {
			return nil, err
		}
		for _, arg := range args {
			for _, match := range inputPlaceholderPattern.FindAllStringSubmatch(arg, -1) {
				required[match[1]] = struct{}{}
			}
		}
		input, ok, err := StdoutPathInput(execution)
		if err != nil {
			return nil, err
		}
		if ok {
			required[input] = struct{}{}
		}
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedKind, execution.Kind)
	}
	names := make([]string, 0, len(required))
	for name := range required {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

// RenderArgs renders CLI args from execution templates and runtime inputs.
func RenderArgs(execution *model.Execution, inputs map[string]any) ([]string, error) {
	if execution == nil {
		return nil, ErrNilExecution
	}
	if execution.Kind != model.ExecutionKindCLI {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedKind, execution.Kind)
	}
	args, err := executionArgs(execution)
	if err != nil {
		return nil, err
	}
	rendered := make([]string, len(args))
	for index, arg := range args {
		rendered[index] = renderArg(arg, inputs)
		if strings.Contains(rendered[index], "{{") || strings.Contains(rendered[index], "}}") {
			return nil, fmt.Errorf("missing input for cli arg template %q", arg)
		}
	}
	return rendered, nil
}

// StdoutPathInput returns the input name whose path should receive stdout.
func StdoutPathInput(execution *model.Execution) (string, bool, error) {
	if execution == nil {
		return "", false, ErrNilExecution
	}
	value, ok := execution.Spec[model.ExecutionSpecStdoutPathInput]
	if !ok {
		return "", false, nil
	}
	if value == nil {
		return "", false, nil
	}
	input, ok := value.(string)
	if !ok {
		return "", false, fmt.Errorf("cli execution stdout path input must be a string")
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return "", false, nil
	}
	if strings.Contains(input, "{{") || strings.Contains(input, "}}") {
		return "", false, fmt.Errorf("cli execution stdout path input must be an input name")
	}
	return input, true, nil
}

func executionArgs(execution *model.Execution) ([]string, error) {
	value, ok := execution.Spec[model.ExecutionSpecArgs]
	if !ok {
		return nil, fmt.Errorf("cli execution args are required")
	}
	return stringSlice(value)
}

func renderArg(arg string, inputs map[string]any) string {
	for key, value := range inputs {
		arg = strings.ReplaceAll(arg, "{{"+key+"}}", fmt.Sprint(value))
	}
	return arg
}

func stringSlice(value any) ([]string, error) {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...), nil
	case []any:
		args := make([]string, len(typed))
		for index, item := range typed {
			arg, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("cli execution args must be strings")
			}
			args[index] = arg
		}
		return args, nil
	default:
		return nil, fmt.Errorf("cli execution args must be a string array")
	}
}
