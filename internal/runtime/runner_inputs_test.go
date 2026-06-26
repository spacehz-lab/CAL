package runtime

import (
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

func TestValidateBindingInputsRequiresExecutionPlaceholders(t *testing.T) {
	binding := core.Binding{
		Execution: core.Execution{
			Kind: core.ExecutionKindCLI,
			Spec: map[string]any{core.ExecutionSpecArgs: []string{"convert", "{{source}}", "--format", "{{format}}"}},
		},
	}

	err := NewRunner(DefaultRegistry()).Validate(binding, map[string]any{"source": "input.docx"})
	if err == nil || !strings.Contains(err.Error(), "missing required input: format") {
		t.Fatalf("Validate() error = %v, want missing format", err)
	}
}

func TestValidateBindingInputsAcceptsAllowedEnum(t *testing.T) {
	binding := constrainedFormatBinding()

	if err := NewRunner(DefaultRegistry()).Validate(binding, map[string]any{"source": "input.docx", "format": "html"}); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateBindingInputsRejectsDisallowedEnum(t *testing.T) {
	binding := constrainedFormatBinding()

	err := NewRunner(DefaultRegistry()).Validate(binding, map[string]any{"source": "input.docx", "format": "pdf"})
	if err == nil || !strings.Contains(err.Error(), `input "format" value "pdf" is not allowed`) {
		t.Fatalf("Validate() error = %v, want rejected format", err)
	}
}

func TestValidateBindingInputsRejectsInvalidConstraintShapes(t *testing.T) {
	for _, test := range []struct {
		name       string
		constraint any
	}{
		{name: "not object", constraint: "bad"},
		{name: "bad enum", constraint: map[string]any{"enum": "bad"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			binding := constrainedFormatBinding()
			binding.InputConstraints["format"] = test.constraint

			err := NewRunner(DefaultRegistry()).Validate(binding, map[string]any{"source": "input.docx", "format": "html"})
			if err == nil {
				t.Fatal("Validate() error = nil, want invalid constraint error")
			}
		})
	}
}

func TestRequiredInputsIncludesStdoutPathInput(t *testing.T) {
	inputs, err := NewRunner(DefaultRegistry()).RequiredInputs(core.Execution{
		Kind: core.ExecutionKindCLI,
		Spec: map[string]any{
			core.ExecutionSpecArgs:            []string{"encode", "{{source}}"},
			core.ExecutionSpecStdoutPathInput: "target",
		},
	})
	if err != nil {
		t.Fatalf("RequiredInputs() error = %v", err)
	}
	if got, want := strings.Join(inputs, ","), "source,target"; got != want {
		t.Fatalf("RequiredInputs() = %q, want %q", got, want)
	}
}

func constrainedFormatBinding() core.Binding {
	return core.Binding{
		InputConstraints: map[string]any{
			"format": map[string]any{
				"type": "string",
				"enum": []any{"html", "json"},
			},
		},
		Execution: core.Execution{
			Kind: core.ExecutionKindCLI,
			Spec: map[string]any{core.ExecutionSpecArgs: []string{"convert", "{{source}}", "--format", "{{format}}"}},
		},
	}
}
