package check

import (
	"fmt"
	"os"
	"strings"

	"github.com/spacehz-lab/cal/internal/model"
)

type checkSubject struct {
	value   any
	path    string
	label   string
	inputs  map[string]any
	outputs map[string]any
}

func resolveSubject(subject *model.VerifySubject, req *Request) (*checkSubject, error) {
	switch subject.Type {
	case model.VerifySubjectFile:
		return pathSubject(subject.Input, req.Inputs)
	case model.VerifySubjectStdout:
		return scalarSubject(string(model.VerifySubjectStdout), req.Stdout, req.Inputs), nil
	case model.VerifySubjectStderr:
		return scalarSubject(string(model.VerifySubjectStderr), req.Stderr, req.Inputs), nil
	case model.VerifySubjectExitCode:
		return scalarSubject(string(model.VerifySubjectExitCode), req.ExitCode, req.Inputs), nil
	default:
		return nil, fmt.Errorf("verify subject type %q is not supported", subject.Type)
	}
}

func pathSubject(input string, inputs map[string]any) (*checkSubject, error) {
	path, ok := inputs[input].(string)
	if !ok || strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("verify subject %q path input is required", input)
	}
	path = strings.TrimSpace(path)
	return &checkSubject{path: path, value: path, label: input, inputs: inputs, outputs: map[string]any{input: path}}, nil
}

func scalarSubject(label string, value any, inputs map[string]any) *checkSubject {
	return &checkSubject{value: value, label: label, inputs: inputs, outputs: map[string]any{label: value}}
}

func subjectText(subject *checkSubject) (string, error) {
	if subject.path == "" {
		return fmt.Sprint(subject.value), nil
	}
	content, err := os.ReadFile(subject.path)
	if err != nil {
		return "", fmt.Errorf("verify subject read: %w", err)
	}
	return string(content), nil
}
