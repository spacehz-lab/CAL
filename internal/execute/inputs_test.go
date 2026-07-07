package execute

import (
	"reflect"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestRequiredInputsReturnsSortedPlaceholdersAndStdoutTarget(t *testing.T) {
	execution := &model.Execution{
		Kind: model.ExecutionKindCLI,
		Spec: map[string]any{
			model.ExecutionSpecArgs:            []string{"run", "{{target}}", "{{source}}", "{{source}}"},
			model.ExecutionSpecStdoutPathInput: "artifact",
		},
	}
	required, err := RequiredInputs(execution)
	if err != nil {
		t.Fatalf("RequiredInputs() error = %v", err)
	}
	want := []string{"artifact", "source", "target"}
	if !reflect.DeepEqual(required, want) {
		t.Fatalf("RequiredInputs() = %#v, want %#v", required, want)
	}
}

func TestRenderArgsRendersStringArgs(t *testing.T) {
	execution := &model.Execution{
		Kind: model.ExecutionKindCLI,
		Spec: map[string]any{model.ExecutionSpecArgs: []string{"convert", "{{source}}", "--format", "{{format}}"}},
	}
	args, err := RenderArgs(execution, map[string]any{"source": "input.md", "format": "pdf"})
	if err != nil {
		t.Fatalf("RenderArgs() error = %v", err)
	}
	want := []string{"convert", "input.md", "--format", "pdf"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("RenderArgs() = %#v, want %#v", args, want)
	}
}

func TestRenderArgsAcceptsJSONDecodedArgs(t *testing.T) {
	execution := &model.Execution{
		Kind: model.ExecutionKindCLI,
		Spec: map[string]any{model.ExecutionSpecArgs: []any{"run", "{{value}}"}},
	}
	args, err := RenderArgs(execution, map[string]any{"value": "ok"})
	if err != nil {
		t.Fatalf("RenderArgs() error = %v", err)
	}
	want := []string{"run", "ok"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("RenderArgs() = %#v, want %#v", args, want)
	}
}

func TestRenderArgsRejectsMissingInput(t *testing.T) {
	execution := &model.Execution{
		Kind: model.ExecutionKindCLI,
		Spec: map[string]any{model.ExecutionSpecArgs: []string{"{{missing}}"}},
	}
	if _, err := RenderArgs(execution, nil); err == nil {
		t.Fatal("RenderArgs() error = nil, want missing input error")
	}
}

func TestRenderArgsRejectsInvalidArgs(t *testing.T) {
	execution := &model.Execution{
		Kind: model.ExecutionKindCLI,
		Spec: map[string]any{model.ExecutionSpecArgs: []any{"run", 1}},
	}
	if _, err := RenderArgs(execution, nil); err == nil {
		t.Fatal("RenderArgs() error = nil, want invalid args error")
	}
}

func TestRequiredInputsRejectsUnsupportedKind(t *testing.T) {
	execution := &model.Execution{Kind: model.ExecutionKindURLOpen}
	if _, err := RequiredInputs(execution); err == nil {
		t.Fatal("RequiredInputs() error = nil, want unsupported kind error")
	}
}

func TestStdoutPathInputRejectsPlaceholder(t *testing.T) {
	execution := &model.Execution{
		Kind: model.ExecutionKindCLI,
		Spec: map[string]any{
			model.ExecutionSpecArgs:            []string{},
			model.ExecutionSpecStdoutPathInput: "{{target}}",
		},
	}
	if _, _, err := StdoutPathInput(execution); err == nil {
		t.Fatal("StdoutPathInput() error = nil, want invalid stdout path input error")
	}
}
