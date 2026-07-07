package surface

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPromptIncludesAcquisitionHint(t *testing.T) {
	req := &Request{Hint: "install a package", MaxItems: 40}

	prompt := prompt(req)

	var payload map[string]any
	if err := json.Unmarshal([]byte(prompt.User), &payload); err != nil {
		t.Fatalf("decode prompt payload: %v", err)
	}
	if payload[promptKeyAcquisitionHint] != "install a package" {
		t.Fatalf("acquisition hint = %#v, want install a package", payload[promptKeyAcquisitionHint])
	}
}

func TestSystemPromptDescribesOptionalUsage(t *testing.T) {
	if !strings.Contains(systemPrompt, `"usage":"optional documented invocation shape"`) {
		t.Fatal("systemPrompt must include usage in the response shape")
	}
	if !strings.Contains(systemPrompt, "usage is optional") {
		t.Fatal("systemPrompt must describe usage as optional")
	}
}
