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
	if !strings.Contains(systemPrompt, "remove the executable name from args") {
		t.Fatal("systemPrompt must tell the model to remove provider executable names from usage")
	}
}

func TestPromptIncludesAcquisitionHintForRuntimeValues(t *testing.T) {
	req := &Request{Hint: "convert a property list file to json"}

	prompt := prompt(req)

	var payload map[string]any
	if err := json.Unmarshal([]byte(prompt.User), &payload); err != nil {
		t.Fatalf("decode prompt payload: %v", err)
	}
	if payload[promptKeyAcquisitionHint] != "convert a property list file to json" {
		t.Fatalf("acquisition hint = %#v, want hint", payload[promptKeyAcquisitionHint])
	}
	if !strings.Contains(systemPrompt, "Use it only to choose concrete runtime-controlled values") {
		t.Fatal("systemPrompt must scope acquisition hint to runtime value selection")
	}
	if !strings.Contains(systemPrompt, "instead of choosing an arbitrary default") {
		t.Fatal("systemPrompt must prefer hinted runtime values over arbitrary defaults")
	}
}

func TestSystemPromptSupportsPositionalAndStdoutBindings(t *testing.T) {
	if !strings.Contains(systemPrompt, "When selected_surface_items is non-empty, candidates must be non-empty") {
		t.Fatal("systemPrompt must make non-empty candidates a hard requirement for selected surfaces")
	}
	if !strings.Contains(systemPrompt, `Do not output {"candidates":[],"probe_material":[]}`) {
		t.Fatal("systemPrompt must explicitly reject empty candidate arrays for selected surfaces")
	}
	if !strings.Contains(systemPrompt, "Binding is not a filter stage") {
		t.Fatal("systemPrompt must not make binding filter selected surfaces")
	}
	if !strings.Contains(systemPrompt, "return at least one best-effort candidate") {
		t.Fatal("systemPrompt must require a candidate for non-empty selected surfaces with invocation details")
	}
	if !strings.Contains(systemPrompt, "positional operands") {
		t.Fatal("systemPrompt must not allow empty candidates only because invocation uses positional operands")
	}
	if !strings.Contains(systemPrompt, "Map documented input file, path, source, or FILE operands to {{source}}") {
		t.Fatal("systemPrompt must describe generic positional input operand mapping")
	}
	if !strings.Contains(systemPrompt, "Stdout is a valid primary output") {
		t.Fatal("systemPrompt must treat stdout as a valid binding output")
	}
	if !strings.Contains(systemPrompt, `must not name an input source such as "source", "input", or "stdin"`) {
		t.Fatal("systemPrompt must prevent stdout_path_input from pointing to input sources")
	}
	if strings.Contains(systemPrompt, "later verification needs") {
		t.Fatal("systemPrompt must not make binding infer later evidence needs")
	}
	if strings.Contains(systemPrompt, "can be mapped to CLI args") {
		t.Fatal("systemPrompt must not leave subjective can-be-mapped filtering language")
	}
}

func TestSystemPromptScopesObservationsToSelectedSurfaces(t *testing.T) {
	if !strings.Contains(systemPrompt, "Use selected_surface_items as the primary source") {
		t.Fatal("systemPrompt must make selected surfaces primary")
	}
	if !strings.Contains(systemPrompt, "Use observations only to recover missing invocation details") {
		t.Fatal("systemPrompt must scope observations to missing invocation details")
	}
	if !strings.Contains(systemPrompt, "default behavior") {
		t.Fatal("systemPrompt must allow observations to recover selected-surface default behavior")
	}
	if !strings.Contains(systemPrompt, "Do not use acquisition_hint or observations to discover unrelated surfaces") {
		t.Fatal("systemPrompt must prevent hint/observation-driven reselection")
	}
	if strings.Contains(systemPrompt, "supports the current capability") || strings.Contains(systemPrompt, "usage for the planned capability") {
		t.Fatal("systemPrompt must not ask binding to re-evaluate capability semantics")
	}
	if !strings.Contains(systemPrompt, "may be narrower than capability_plan.description") {
		t.Fatal("systemPrompt must allow execution descriptions to be narrower than the plan")
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
	if _, ok := payload["observations"]; !ok {
		t.Fatalf("payload keys = %#v, want observations for supporting context", payload)
	}
	if _, ok := payload["surface_items"]; ok {
		t.Fatalf("payload contains legacy surface_items key: %#v", payload)
	}
}
