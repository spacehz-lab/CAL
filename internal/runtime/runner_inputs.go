package runtime

import (
	"fmt"
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

func executionArgs(execution core.Execution) ([]string, error) {
	value, ok := execution.Spec[core.ExecutionSpecArgs]
	if !ok {
		return nil, fmt.Errorf("cli execution args are required")
	}
	return stringSlice(value)
}
