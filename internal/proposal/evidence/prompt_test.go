package evidence

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPromptIncludesVerifyPredicateRules(t *testing.T) {
	prompt := prompt(&Request{})

	var payload map[string]any
	if err := json.Unmarshal([]byte(prompt.User), &payload); err != nil {
		t.Fatalf("decode prompt payload: %v", err)
	}
	if _, ok := payload[promptKeyVerifyPredicateRules]; !ok {
		t.Fatalf("payload keys = %#v, want %q", payload, promptKeyVerifyPredicateRules)
	}
}

func TestSystemPromptUsesVerifyPredicateRules(t *testing.T) {
	if !strings.Contains(systemPrompt, "Use verify_predicate_rules for predicate params") {
		t.Fatal("systemPrompt must point the model at verify_predicate_rules")
	}
}

func TestSystemPromptAvoidsWhitespaceSensitiveJSONContains(t *testing.T) {
	if !strings.Contains(systemPrompt, "do not use whitespace-sensitive contains checks") {
		t.Fatal("systemPrompt must avoid whitespace-sensitive JSON contains checks")
	}
	if !strings.Contains(systemPrompt, "contains_any with compact and spaced forms") {
		t.Fatal("systemPrompt must allow robust JSON content checks")
	}
}
