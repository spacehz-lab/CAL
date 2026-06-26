package use

import (
	"context"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
	sharedllm "github.com/spacehz-lab/cal/internal/llm"
)

func TestResolverUsesLLMSelectorForMultipleCandidates(t *testing.T) {
	client := &fakeLLMClient{content: []byte(`{"binding_id":"binding_resize","reason":"best semantic match"}`)}
	capabilities := []core.Capability{
		testCapability("document.export_pdf", "Export a document to PDF.", "binding_pdf", "provider_a", []string{"export-pdf"}),
		testCapability("image.resize", "Resize an image.", "binding_resize", "provider_b", []string{"resize", "{{source}}", "{{target}}"}),
	}

	selection, err := NewResolver(Request{
		Intent: "export or resize image",
		Inputs: map[string]any{
			"source": "/private/source.png",
			"target": "/private/target.png",
		},
	}, WithLLM(client)).Select(context.Background(), capabilities)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if selection.Source != selectionSourceLLM || selection.BindingID != "binding_resize" || selection.Reason != "best semantic match" {
		t.Fatalf("Select() = %#v, want llm resize selection", selection)
	}
	if !strings.Contains(client.prompt.User, `"input_keys":["source","target"]`) {
		t.Fatalf("prompt user = %q, want input keys", client.prompt.User)
	}
	if strings.Contains(client.prompt.User, "/private/source.png") || strings.Contains(client.prompt.User, "/private/target.png") {
		t.Fatalf("prompt user = %q, want no raw input values", client.prompt.User)
	}
}

func TestResolverUsesLLMSelectorForInputPatch(t *testing.T) {
	client := &fakeLLMClient{content: []byte(`{"binding_id":"binding_pdf","inputs_patch":{"source":"/tmp/source.txt"},"reason":"source path appears in intent"}`)}
	capabilities := []core.Capability{
		testCapability("document.export_pdf", "Export a document to PDF.", "binding_pdf", "provider_a", []string{"export-pdf", "--source", "{{source}}", "--target", "{{target}}"}),
	}

	resolution, err := NewResolver(Request{
		Intent: "export /tmp/source.txt as pdf",
	}, WithLLM(client)).Resolve(context.Background(), capabilities)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolution.Selection.BindingID != "binding_pdf" || resolution.Selection.inputsPatch["source"] != "/tmp/source.txt" {
		t.Fatalf("Resolve() = %#v, want binding_pdf with source patch", resolution)
	}
	if !strings.Contains(client.prompt.System, "inputs_patch") || !strings.Contains(client.prompt.System, "Do not include target") {
		t.Fatalf("prompt system = %q, want inputs_patch rules", client.prompt.System)
	}
}

func TestResolverRejectsUnknownLLMInputPatch(t *testing.T) {
	client := &fakeLLMClient{content: []byte(`{"binding_id":"binding_pdf","inputs_patch":{"extra":"value"}}`)}
	capabilities := []core.Capability{
		testCapability("document.export_pdf", "Export a document to PDF.", "binding_pdf", "provider_a", []string{"export-pdf", "--source", "{{source}}"}),
	}

	_, err := NewResolver(Request{
		Intent: "export /tmp/source.txt as pdf",
	}, WithLLM(client)).Resolve(context.Background(), capabilities)
	if err == nil || err.Code != CodeInvalidLLMSelection {
		t.Fatalf("Resolve() error = %#v, want invalid llm selection", err)
	}
}

func TestResolverIgnoresDuplicateLLMInputPatch(t *testing.T) {
	client := &fakeLLMClient{content: []byte(`{"binding_id":"binding_pdf","inputs_patch":{"format":"html"}}`)}
	capabilities := []core.Capability{
		testCapability("document.convert_format", "Convert a document to a specified format.", "binding_pdf", "provider_a", []string{"convert", "{{source}}", "{{format}}", "{{target}}"}),
		testCapability("document.convert_text", "Convert a document to text.", "binding_text", "provider_a", []string{"convert-text", "{{source}}", "{{target}}"}),
	}

	resolution, err := NewResolver(Request{
		Intent: "convert document to html",
		Inputs: map[string]any{
			"source": "/tmp/source.txt",
			"format": "html",
		},
	}, WithLLM(client)).Resolve(context.Background(), capabilities)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if len(resolution.Selection.inputsPatch) != 0 {
		t.Fatalf("inputsPatch = %#v, want duplicate patch ignored", resolution.Selection.inputsPatch)
	}
}

func TestResolverRejectsConflictingDuplicateLLMInputPatch(t *testing.T) {
	client := &fakeLLMClient{content: []byte(`{"binding_id":"binding_pdf","inputs_patch":{"format":"txt"}}`)}
	capabilities := []core.Capability{
		testCapability("document.convert_format", "Convert a document to a specified format.", "binding_pdf", "provider_a", []string{"convert", "{{source}}", "{{format}}", "{{target}}"}),
		testCapability("document.convert_text", "Convert a document to text.", "binding_text", "provider_a", []string{"convert-text", "{{source}}", "{{target}}"}),
	}

	_, err := NewResolver(Request{
		Intent: "convert document to html",
		Inputs: map[string]any{
			"source": "/tmp/source.txt",
			"format": "html",
		},
	}, WithLLM(client)).Resolve(context.Background(), capabilities)
	if err == nil || err.Code != CodeInvalidLLMSelection {
		t.Fatalf("Resolve() error = %#v, want invalid llm selection", err)
	}
}

func TestResolverRejectsUnknownLLMSelection(t *testing.T) {
	client := &fakeLLMClient{content: []byte(`{"binding_id":"binding_missing"}`)}
	capabilities := []core.Capability{
		testCapability("document.export_pdf", "Export a document to PDF.", "binding_pdf", "provider_a", []string{"export-pdf"}),
		testCapability("image.resize", "Resize an image.", "binding_resize", "provider_b", []string{"resize"}),
	}

	_, err := NewResolver(Request{
		Intent: "export or resize image",
		Inputs: map[string]any{},
	}, WithLLM(client)).Select(context.Background(), capabilities)
	if err == nil || err.Code != CodeInvalidLLMSelection {
		t.Fatalf("Select() error = %#v, want invalid llm selection", err)
	}
}

type fakeLLMClient struct {
	content []byte
	err     error
	prompt  sharedllm.Prompt
}

func (client *fakeLLMClient) Complete(_ context.Context, prompt sharedllm.Prompt) ([]byte, error) {
	client.prompt = prompt
	if client.err != nil {
		return nil, client.err
	}
	return client.content, nil
}
