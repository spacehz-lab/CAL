package verify

import (
	"fmt"
	"os"
	"strings"

	"github.com/spacehz-lab/cal/internal/core"
)

type checkSubject struct {
	value   any
	path    string
	label   string
	inputs  map[string]any
	outputs map[string]any
}

func evaluateSubject(subject core.VerifySubject, verifyContext Context) (checkSubject, error) {
	switch subject.Type {
	case core.VerifySubjectFile:
		return pathSubject(subject.Input, verifyContext.Inputs)
	case core.VerifySubjectStdout:
		return scalarSubject(string(core.VerifySubjectStdout), verifyContext.Stdout, verifyContext.Inputs)
	case core.VerifySubjectStderr:
		return scalarSubject(string(core.VerifySubjectStderr), verifyContext.Stderr, verifyContext.Inputs)
	case core.VerifySubjectExitCode:
		return scalarSubject(string(core.VerifySubjectExitCode), verifyContext.ExitCode, verifyContext.Inputs)
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
