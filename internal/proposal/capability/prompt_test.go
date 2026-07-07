package capability

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPromptIncludesAcquisitionHint(t *testing.T) {
	req := &Request{Hint: "install a package", MaxPlans: 3}

	prompt := prompt(req)

	var payload map[string]any
	if err := json.Unmarshal([]byte(prompt.User), &payload); err != nil {
		t.Fatalf("decode prompt payload: %v", err)
	}
	if payload[promptKeyAcquisitionHint] != "install a package" {
		t.Fatalf("acquisition hint = %#v, want install a package", payload[promptKeyAcquisitionHint])
	}
}

func TestSystemPromptNarrowsOnAcquisitionHint(t *testing.T) {
	if !strings.Contains(systemPrompt, "treat it as a narrowing constraint") {
		t.Fatal("systemPrompt must treat acquisition_hint as a narrowing constraint")
	}
	if !strings.Contains(systemPrompt, "If acquisition_hint is absent, keep broad discovery") {
		t.Fatal("systemPrompt must preserve broad discovery without hint")
	}
}
