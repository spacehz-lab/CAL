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

func TestSystemPromptPrefersStructuredPredicatesForStructuredFormats(t *testing.T) {
	if !strings.Contains(systemPrompt, "For structured formats, prefer predicates that parse the structure when available") {
		t.Fatal("systemPrompt must prefer structured predicates for structured formats")
	}
	if !strings.Contains(systemPrompt, "Use regex only for stable text formats") {
		t.Fatal("systemPrompt must avoid making regex the first choice for structured fields")
	}
}
