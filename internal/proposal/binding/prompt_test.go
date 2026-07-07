package binding

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSystemPromptAvoidsPlaceholderProviderID(t *testing.T) {
	if strings.Contains(systemPrompt, `"provider_id":"optional"`) {
		t.Fatal("systemPrompt must not include placeholder provider_id example")
	}
}

func TestSystemPromptRequiresProbeMaterialPerCandidate(t *testing.T) {
	if !strings.Contains(systemPrompt, "Every candidate must have exactly one probe_material record") {
		t.Fatal("systemPrompt must require one probe_material record per candidate")
	}
	if !strings.Contains(systemPrompt, `"inputs":{}`) {
		t.Fatal("systemPrompt must tell the model to include empty inputs for no-placeholder candidates")
	}
}

func TestSystemPromptUsesSelectedSurfaceUsage(t *testing.T) {
	if !strings.Contains(systemPrompt, "selected_surface_items are the surfaces chosen by Capability") {
		t.Fatal("systemPrompt must describe selected surface ownership")
	}
	if !strings.Contains(systemPrompt, "usage as the strongest invocation hint") {
		t.Fatal("systemPrompt must prioritize selected surface usage")
	}
}

func TestPromptUsesSelectedSurfaceItemsPayload(t *testing.T) {
	req := &Request{
		Surfaces: []SurfaceItem{{ID: "s1", Name: "make-pdf", Usage: "make-pdf --in <input> --out <output>"}},
	}

	prompt := prompt(req)

	var payload map[string]any
	if err := json.Unmarshal([]byte(prompt.User), &payload); err != nil {
		t.Fatalf("decode prompt payload: %v", err)
	}
	if _, ok := payload[promptKeySelectedSurfaceItems]; !ok {
		t.Fatalf("payload keys = %#v, want %q", payload, promptKeySelectedSurfaceItems)
	}
	if _, ok := payload["surface_items"]; ok {
		t.Fatalf("payload contains legacy surface_items key: %#v", payload)
	}
}
