package config

import (
	"fmt"
	"os"
	"strings"
)

const (
	// LLMAPIResponses selects the OpenAI Responses API.
	LLMAPIResponses = "responses"
	// LLMAPIChatCompletions selects an OpenAI-compatible Chat Completions API.
	LLMAPIChatCompletions = "chat_completions"

	EnvLLMAPI     = "CAL_LLM_API"
	EnvLLMBaseURL = "CAL_LLM_BASE_URL"
	EnvLLMModel   = "CAL_LLM_MODEL"
	EnvLLMAPIKey  = "CAL_LLM_API_KEY"

	apiKeyRefEnvPrefix = "env:"
)

// LLMSettings contains durable non-secret LLM settings.
type LLMSettings struct {
	API       string `json:"api,omitempty"`
	BaseURL   string `json:"base_url,omitempty"`
	Model     string `json:"model,omitempty"`
	APIKeyRef string `json:"api_key_ref,omitempty"`
}

// LLMConfig is runtime-only LLM configuration.
type LLMConfig struct {
	API     string
	BaseURL string
	Model   string
	APIKey  string
}

// LLMFromEnv reads explicit runtime LLM configuration.
func LLMFromEnv() LLMConfig {
	return LLMConfig{
		API:     strings.TrimSpace(os.Getenv(EnvLLMAPI)),
		BaseURL: strings.TrimSpace(os.Getenv(EnvLLMBaseURL)),
		Model:   strings.TrimSpace(os.Getenv(EnvLLMModel)),
		APIKey:  strings.TrimSpace(os.Getenv(EnvLLMAPIKey)),
	}
}

// Empty reports whether no LLM environment configuration was supplied.
func (cfg LLMConfig) Empty() bool {
	return cfg.API == "" && cfg.BaseURL == "" && cfg.Model == "" && cfg.APIKey == ""
}

// RuntimeLLMConfig resolves durable LLM settings with explicit environment overrides.
func (cfg Config) RuntimeLLMConfig() (LLMConfig, error) {
	var resolved LLMConfig
	if cfg.LLM != nil {
		resolved = LLMConfig{
			API:     strings.TrimSpace(cfg.LLM.API),
			BaseURL: strings.TrimSpace(cfg.LLM.BaseURL),
			Model:   strings.TrimSpace(cfg.LLM.Model),
		}
	}

	env := LLMFromEnv()
	if env.API != "" {
		resolved.API = env.API
	}
	if env.BaseURL != "" {
		resolved.BaseURL = env.BaseURL
	}
	if env.Model != "" {
		resolved.Model = env.Model
	}
	if env.APIKey != "" {
		resolved.APIKey = env.APIKey
		return resolved, nil
	}
	if cfg.LLM == nil || strings.TrimSpace(cfg.LLM.APIKeyRef) == "" {
		return resolved, nil
	}

	apiKey, err := resolveAPIKeyRef(cfg.LLM.APIKeyRef)
	if err != nil {
		return LLMConfig{}, err
	}
	resolved.APIKey = apiKey
	return resolved, nil
}

func resolveAPIKeyRef(ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", nil
	}
	name, ok := strings.CutPrefix(ref, apiKeyRefEnvPrefix)
	if !ok {
		return "", fmt.Errorf("unsupported llm api key ref %q", ref)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("llm api key env ref is required")
	}
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return "", fmt.Errorf("llm api key env ref %q is not set", name)
	}
	return value, nil
}
