package config

import "testing"

func TestLLMFromEnvReadsExplicitConfig(t *testing.T) {
	t.Setenv(EnvLLMAPI, " chat_completions ")
	t.Setenv(EnvLLMBaseURL, " https://api.example.test/v1 ")
	t.Setenv(EnvLLMModel, " test-model ")
	t.Setenv(EnvLLMAPIKey, " test-key ")

	cfg := LLMFromEnv()
	if cfg.API != LLMAPIChatCompletions || cfg.BaseURL != "https://api.example.test/v1" || cfg.Model != "test-model" || cfg.APIKey != "test-key" {
		t.Fatalf("LLMFromEnv() = %#v, want trimmed explicit config", cfg)
	}
	if cfg.Empty() {
		t.Fatal("LLMFromEnv().Empty() = true, want false")
	}
}

func TestLLMFromEnvIgnoresVendorKeys(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "openai-key")
	t.Setenv("MOONSHOT_API_KEY", "moonshot-key")

	cfg := LLMFromEnv()
	if !cfg.Empty() {
		t.Fatalf("LLMFromEnv() = %#v, want empty config without CAL_LLM_*", cfg)
	}
}

func TestRuntimeLLMConfigUsesDurableSettingsAndEnvRef(t *testing.T) {
	t.Setenv("CAL_TEST_LLM_API_KEY", " test-key ")

	cfg, err := Config{
		LLM: &LLMSettings{
			API:       " chat_completions ",
			BaseURL:   " https://api.example.test/v1 ",
			Model:     " test-model ",
			APIKeyRef: "env:CAL_TEST_LLM_API_KEY",
		},
	}.RuntimeLLMConfig()
	if err != nil {
		t.Fatalf("RuntimeLLMConfig() error = %v", err)
	}
	if cfg.API != LLMAPIChatCompletions || cfg.BaseURL != "https://api.example.test/v1" || cfg.Model != "test-model" || cfg.APIKey != "test-key" {
		t.Fatalf("RuntimeLLMConfig() = %#v, want durable settings plus env-ref key", cfg)
	}
}

func TestRuntimeLLMConfigEnvOverridesDurableSettings(t *testing.T) {
	t.Setenv(EnvLLMAPI, LLMAPIResponses)
	t.Setenv(EnvLLMBaseURL, "https://env.example.test/v1")
	t.Setenv(EnvLLMModel, "env-model")
	t.Setenv(EnvLLMAPIKey, "env-key")
	t.Setenv("CAL_TEST_LLM_API_KEY", "config-key")

	cfg, err := Config{
		LLM: &LLMSettings{
			API:       LLMAPIChatCompletions,
			BaseURL:   "https://config.example.test/v1",
			Model:     "config-model",
			APIKeyRef: "env:CAL_TEST_LLM_API_KEY",
		},
	}.RuntimeLLMConfig()
	if err != nil {
		t.Fatalf("RuntimeLLMConfig() error = %v", err)
	}
	if cfg.API != LLMAPIResponses || cfg.BaseURL != "https://env.example.test/v1" || cfg.Model != "env-model" || cfg.APIKey != "env-key" {
		t.Fatalf("RuntimeLLMConfig() = %#v, want explicit env override", cfg)
	}
}

func TestRuntimeLLMConfigRejectsUnsupportedAPIKeyRef(t *testing.T) {
	_, err := Config{
		LLM: &LLMSettings{
			APIKeyRef: "keychain:cal/default/llm",
		},
	}.RuntimeLLMConfig()
	if err == nil {
		t.Fatal("RuntimeLLMConfig() error = nil, want unsupported ref error")
	}
}

func TestRuntimeLLMConfigRejectsMissingEnvRef(t *testing.T) {
	_, err := Config{
		LLM: &LLMSettings{
			APIKeyRef: "env:CAL_MISSING_LLM_API_KEY",
		},
	}.RuntimeLLMConfig()
	if err == nil {
		t.Fatal("RuntimeLLMConfig() error = nil, want missing env ref error")
	}
}
