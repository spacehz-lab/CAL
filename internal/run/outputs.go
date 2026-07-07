package run

import (
	"github.com/spacehz-lab/cal/internal/check"
	"github.com/spacehz-lab/cal/internal/execute"
	"github.com/spacehz-lab/cal/internal/model"
)

func checkRequest(spec *model.VerifySpec, inputs map[string]any, outputs execute.Outputs) *check.Request {
	return &check.Request{
		Spec:     spec,
		Inputs:   inputs,
		Stdout:   textOutput(outputs, execute.OutputStdout),
		Stderr:   textOutput(outputs, execute.OutputStderr),
		ExitCode: numberOutput(outputs, execute.OutputExitCode),
	}
}

func durableOutputs(outputs execute.Outputs) map[string]any {
	if len(outputs) == 0 {
		return nil
	}
	values := map[string]any{}
	for name, output := range outputs {
		switch output.Kind {
		case execute.OutputKindText:
			values[string(name)] = output.Text
		case execute.OutputKindNumber:
			if output.Number != nil {
				values[string(name)] = *output.Number
			}
		case execute.OutputKindFile:
			values[string(name)] = output.Path
		case execute.OutputKindLink:
			values[string(name)] = output.Text
		}
	}
	return values
}

func mergeOutputs(executionOutputs execute.Outputs, checked map[string]any) map[string]any {
	values := durableOutputs(executionOutputs)
	if values == nil {
		values = map[string]any{}
	}
	for key, value := range checked {
		values[key] = value
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

func nonZeroExit(outputs execute.Outputs) bool {
	output, ok := outputs[execute.OutputExitCode]
	if !ok || output.Number == nil {
		return false
	}
	return *output.Number != 0
}

func textOutput(outputs execute.Outputs, name execute.OutputName) string {
	output, ok := outputs[name]
	if !ok {
		return ""
	}
	return output.Text
}

func numberOutput(outputs execute.Outputs, name execute.OutputName) int {
	output, ok := outputs[name]
	if !ok || output.Number == nil {
		return 0
	}
	return *output.Number
}
