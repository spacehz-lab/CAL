package config

import (
	"fmt"
	"strings"
)

// LLMAPI identifies the durable LLM API mode.
type LLMAPI string

const (
	LLMAPIResponses       LLMAPI = "responses"
	LLMAPIChatCompletions LLMAPI = "chat_completions"

	apiKeyRefEnvPrefix = "env:"
)

// LLMConfig contains durable non-secret LLM settings.
type LLMConfig struct {
	API       LLMAPI `json:"api,omitempty"`
	BaseURL   string `json:"base_url,omitempty"`
	Model     string `json:"model,omitempty"`
	APIKeyRef string `json:"api_key_ref,omitempty"`
}

// Validate checks durable LLM settings without resolving secrets.
func (cfg *LLMConfig) Validate() error {
	if cfg == nil {
		return nil
	}
	if cfg.API != "" && !validLLMAPI(cfg.API) {
		return fmt.Errorf("unsupported llm api %q", cfg.API)
	}
	if cfg.APIKeyRef != "" && !strings.HasPrefix(cfg.APIKeyRef, apiKeyRefEnvPrefix) {
		return fmt.Errorf("unsupported llm api key ref %q", cfg.APIKeyRef)
	}
	if cfg.APIKeyRef == apiKeyRefEnvPrefix {
		return fmt.Errorf("llm api key env ref is required")
	}
	return nil
}

func (cfg *LLMConfig) withDefaults() {
	cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
	cfg.Model = strings.TrimSpace(cfg.Model)
	cfg.APIKeyRef = strings.TrimSpace(cfg.APIKeyRef)
	cfg.API = LLMAPI(strings.TrimSpace(string(cfg.API)))
}

func validLLMAPI(api LLMAPI) bool {
	switch api {
	case LLMAPIResponses, LLMAPIChatCompletions:
		return true
	default:
		return false
	}
}
