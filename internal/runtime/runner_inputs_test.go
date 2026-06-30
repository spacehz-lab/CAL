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
