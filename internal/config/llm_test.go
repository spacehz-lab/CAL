package config

import "testing"

func TestLLMConfigAllowsEmptyConfig(t *testing.T) {
	var cfg LLMConfig
	cfg.withDefaults()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestLLMConfigAllowsSupportedAPIs(t *testing.T) {
	for _, api := range []LLMAPI{LLMAPIResponses, LLMAPIChatCompletions} {
		cfg := LLMConfig{API: api}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("Validate(%q) error = %v", api, err)
		}
	}
}

func TestLLMConfigRejectsUnsupportedAPI(t *testing.T) {
	cfg := LLMConfig{API: "legacy"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want unsupported api error")
	}
}

func TestLLMConfigValidatesAPIKeyRefWithoutResolving(t *testing.T) {
	cfg := LLMConfig{APIKeyRef: " env:CAL_TEST_LLM_API_KEY "}
	cfg.withDefaults()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if cfg.APIKeyRef != "env:CAL_TEST_LLM_API_KEY" {
		t.Fatalf("APIKeyRef = %q, want trimmed ref", cfg.APIKeyRef)
	}
}

func TestLLMConfigRejectsUnsupportedAPIKeyRef(t *testing.T) {
	cfg := LLMConfig{APIKeyRef: "keychain:cal/default/llm"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want unsupported api key ref error")
	}
}
