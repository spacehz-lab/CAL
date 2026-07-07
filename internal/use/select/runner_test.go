package selector

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/llm"
	"github.com/spacehz-lab/cal/internal/model"
)

func TestRunSelectsLocalCandidate(t *testing.T) {
	result, err := NewRunner().Run(context.Background(), &Request{
		Intent:       "convert markdown to pdf",
		Inputs:       map[string]any{"source": "input.md"},
		Capabilities: []model.Capability{capability("markdown_to_pdf", "Convert Markdown to PDF", binding("binding_pdf", "provider_a", model.VerifyLevelL2, []string{"run", "{{source}}"}))},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Source != SourceLocal || result.BindingID != "binding_pdf" {
		t.Fatalf("result = %#v, want local binding_pdf", result)
	}
}

func TestRunReturnsNoMatch(t *testing.T) {
	_, err := NewRunner().Run(context.Background(), &Request{
		Intent:       "resize image",
		Capabilities: []model.Capability{capability("markdown_to_pdf", "Convert Markdown to PDF", binding("binding_pdf", "provider_a", model.VerifyLevelL2, []string{"run"}))},
	})
	var selectionErr *Error
	if !errors.As(err, &selectionErr) || selectionErr.Code != CodeNoMatch {
		t.Fatalf("Run() error = %v, want no_match", err)
	}
}

func TestRunReturnsAmbiguousLocalSelection(t *testing.T) {
	_, err := NewRunner().Run(context.Background(), &Request{
		Intent: "export file",
		Capabilities: []model.Capability{
			capability("export_pdf", "Export file", binding("binding_pdf", "provider_a", model.VerifyLevelL2, []string{"run"})),
			capability("export_png", "Export file", binding("binding_png", "provider_b", model.VerifyLevelL2, []string{"run"})),
		},
	})
	var selectionErr *Error
	if !errors.As(err, &selectionErr) || selectionErr.Code != CodeAmbiguous {
		t.Fatalf("Run() error = %v, want ambiguous", err)
	}
}

func TestRunFiltersByDefaultL2VerifyLevel(t *testing.T) {
	result, err := NewRunner().Run(context.Background(), &Request{
		Intent: "export file",
		Capabilities: []model.Capability{
			capability("export_file_l1", "Export file", binding("binding_l1", "provider_a", model.VerifyLevelL1, []string{"run"})),
			capability("export_file_l2", "Export file", binding("binding_l2", "provider_b", model.VerifyLevelL2, []string{"run"})),
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.BindingID != "binding_l2" {
		t.Fatalf("binding id = %q, want binding_l2", result.BindingID)
	}
}

func TestRunUsesLLMForTiedCandidates(t *testing.T) {
	client := &fakeLLM{response: `{"binding_id":"binding_png","reason":"best visual export"}`}
	result, err := NewRunner(WithLLM(client)).Run(context.Background(), &Request{
		Intent: "export file",
		Capabilities: []model.Capability{
			capability("export_pdf", "Export file", binding("binding_pdf", "provider_a", model.VerifyLevelL2, []string{"run"})),
			capability("export_png", "Export file", binding("binding_png", "provider_b", model.VerifyLevelL2, []string{"run"})),
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Source != SourceLLM || result.BindingID != "binding_png" {
		t.Fatalf("result = %#v, want LLM binding_png", result)
	}
	if client.request == nil || !client.request.JSON {
		t.Fatalf("llm request = %#v, want JSON request", client.request)
	}
}

func TestRunRejectsLLMUnknownBinding(t *testing.T) {
	client := &fakeLLM{response: `{"binding_id":"missing"}`}
	_, err := NewRunner(WithLLM(client)).Run(context.Background(), &Request{
		Intent: "export file",
		Capabilities: []model.Capability{
			capability("export_pdf", "Export file", binding("binding_pdf", "provider_a", model.VerifyLevelL2, []string{"run"})),
			capability("export_png", "Export file", binding("binding_png", "provider_b", model.VerifyLevelL2, []string{"run"})),
		},
	})
	var selectionErr *Error
	if !errors.As(err, &selectionErr) || selectionErr.Code != CodeInvalidLLMSelection {
		t.Fatalf("Run() error = %v, want invalid LLM selection", err)
	}
}

func TestRunRejectsLLMInputsPatchOverwrite(t *testing.T) {
	client := &fakeLLM{response: `{"binding_id":"binding_pdf","inputs_patch":{"source":"other.md"}}`}
	_, err := NewRunner(WithLLM(client)).Run(context.Background(), &Request{
		Intent: "export file",
		Inputs: map[string]any{"source": "input.md"},
		Capabilities: []model.Capability{
			capability("export_pdf", "Export file", binding("binding_pdf", "provider_a", model.VerifyLevelL2, []string{"run", "{{source}}"})),
			capability("export_png", "Export file", binding("binding_png", "provider_b", model.VerifyLevelL2, []string{"run", "{{source}}"})),
		},
	})
	var selectionErr *Error
	if !errors.As(err, &selectionErr) || selectionErr.Code != CodeInvalidLLMSelection {
		t.Fatalf("Run() error = %v, want invalid LLM selection", err)
	}
}

func TestLLMSystemPromptContainsBoundedSelectionRules(t *testing.T) {
	required := []string{
		"Choose only from candidates[].binding_id.",
		"Return exactly one JSON object:",
		"inputs_patch may include only keys required by the selected candidate.",
		"Do not include target in inputs_patch",
		"Do not invent capabilities, bindings, inputs, commands, files, paths, formats, algorithms, or outcomes.",
		"Do not include markdown, comments, or extra text.",
	}
	for _, text := range required {
		if !strings.Contains(llmSystemPrompt, text) {
			t.Fatalf("llmSystemPrompt missing %q", text)
		}
	}
}

type fakeLLM struct {
	request  *llm.Request
	response string
	err      error
}

func (client *fakeLLM) Model() string {
	return "fake"
}

func (client *fakeLLM) Complete(_ context.Context, req *llm.Request) (*llm.Response, error) {
	client.request = req
	if client.err != nil {
		return nil, client.err
	}
	return &llm.Response{Text: client.response, Model: "fake"}, nil
}

func capability(id string, description string, bindings ...model.Binding) model.Capability {
	return model.Capability{ID: id, Description: description, Bindings: bindings}
}

func binding(id string, providerID string, level model.VerifyLevel, args []string) model.Binding {
	return model.Binding{
		ID:           id,
		CapabilityID: "",
		ProviderID:   providerID,
		Execution:    model.Execution{Kind: model.ExecutionKindCLI, Spec: map[string]any{model.ExecutionSpecArgs: args}},
		Verify:       verifySpec(level),
		Evidence:     []model.EvidenceRef{{ID: "evidence_1"}},
		State:        model.BindingStatePromoted,
	}
}

func verifySpec(level model.VerifyLevel) *model.VerifySpec {
	return &model.VerifySpec{
		Level:  level,
		Method: model.VerifyMethodExecute,
		Checks: []model.VerifyCheck{{
			Subject:   model.VerifySubject{Type: model.VerifySubjectStdout},
			Predicate: model.VerifyPredicateNonEmpty,
		}},
	}
}
