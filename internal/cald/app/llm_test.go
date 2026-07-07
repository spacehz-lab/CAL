package app

import (
	"errors"
	"testing"

	"github.com/spacehz-lab/cal/internal/config"
	"github.com/spacehz-lab/cal/internal/llm"
)

func TestLLMOptionsUsesExplicitRuntimeOptionsFirst(t *testing.T) {
	t.Setenv(envLLMAPI, string(llm.APIResponses))
	t.Setenv(envLLMModel, "env-model")
	t.Setenv(envLLMAPIKey, "env-key")

	runtime := &llm.Options{API: llm.APIChatCompletions, Model: "runtime-model", APIKey: "runtime-key"}
	options, ok, err := llmOptions(runtime, nil)
	if err != nil {
		t.Fatalf("llmOptions() error = %v", err)
	}
	if !ok || options != runtime {
		t.Fatalf("llmOptions() = %#v, %v; want explicit runtime options", options, ok)
	}
}

func TestLLMOptionsFromEnvironment(t *testing.T) {
	t.Setenv(envLLMAPI, string(llm.APIChatCompletions))
	t.Setenv(envLLMBaseURL, " https://llm.example.test/v1 ")
	t.Setenv(envLLMModel, " test-model ")
	t.Setenv(envLLMAPIKey, " test-key ")

	options, ok, err := llmOptions(nil, nil)
	if err != nil {
		t.Fatalf("llmOptions() error = %v", err)
	}
	if !ok {
		t.Fatal("llmOptions() ok = false, want true")
	}
	if options.API != llm.APIChatCompletions || options.BaseURL != "https://llm.example.test/v1" || options.Model != "test-model" || options.APIKey != "test-key" {
		t.Fatalf("options = %#v, want trimmed env options", options)
	}
}

func TestLLMOptionsEnvironmentOverridesConfig(t *testing.T) {
	t.Setenv("CONFIG_LLM_KEY", "config-key")
	t.Setenv(envLLMAPIKey, "env-key")
	t.Setenv(envLLMModel, "env-model")
	cfg := &config.Config{LLM: &config.LLMConfig{
		API:       config.LLMAPIResponses,
		BaseURL:   "https://config.example.test/v1",
		Model:     "config-model",
		APIKeyRef: "env:CONFIG_LLM_KEY",
	}}

	options, ok, err := llmOptions(nil, cfg)
	if err != nil {
		t.Fatalf("llmOptions() error = %v", err)
	}
	if !ok {
		t.Fatal("llmOptions() ok = false, want true")
	}
	if options.API != llm.APIResponses || options.BaseURL != "https://config.example.test/v1" || options.Model != "env-model" || options.APIKey != "env-key" {
		t.Fatalf("options = %#v, want env key/model over config", options)
	}
}

func TestLLMOptionsIncompleteEnvironmentReturnsError(t *testing.T) {
	t.Setenv(envLLMAPIKey, "env-key")

	_, ok, err := llmOptions(nil, nil)
	if err == nil {
		t.Fatal("llmOptions() error = nil, want invalid env error")
	}
	if ok {
		t.Fatal("llmOptions() ok = true, want false")
	}
	if !errors.Is(err, llm.ErrMissingAPI) {
		t.Fatalf("llmOptions() error = %v, want missing API", err)
	}
}

func TestLLMOptionsIncompleteConfigWithoutEnvironmentIsDisabled(t *testing.T) {
	cfg := &config.Config{LLM: &config.LLMConfig{
		API:   config.LLMAPIChatCompletions,
		Model: "test-model",
	}}

	options, ok, err := llmOptions(nil, cfg)
	if err != nil {
		t.Fatalf("llmOptions() error = %v", err)
	}
	if ok || options != nil {
		t.Fatalf("llmOptions() = %#v, %v; want disabled missing-key config", options, ok)
	}
}
