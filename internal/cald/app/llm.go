package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/spacehz-lab/cal/internal/config"
	"github.com/spacehz-lab/cal/internal/llm"
)

const (
	envLLMAPI     = "CAL_LLM_API"
	envLLMBaseURL = "CAL_LLM_BASE_URL"
	envLLMModel   = "CAL_LLM_MODEL"
	envLLMAPIKey  = "CAL_LLM_API_KEY"
)

func llmOptions(runtime *llm.Options, cfg *config.Config) (*llm.Options, bool, error) {
	if runtime != nil {
		return runtime, true, nil
	}

	options, hasConfig := llmOptionsFromConfig(cfg)
	hasEnv := applyLLMEnv(options)
	if !hasConfig && !hasEnv {
		return nil, false, nil
	}
	if err := options.Validate(); err != nil {
		if hasEnv {
			return nil, false, fmt.Errorf("invalid CAL_LLM_* environment: %w", err)
		}
		return nil, false, nil
	}
	return options, true, nil
}

func llmOptionsFromConfig(cfg *config.Config) (*llm.Options, bool) {
	options := &llm.Options{}
	if cfg == nil || cfg.LLM == nil {
		return options, false
	}
	options.API = llm.API(cfg.LLM.API)
	options.BaseURL = cfg.LLM.BaseURL
	options.Model = cfg.LLM.Model
	if apiKey, ok := resolveAPIKey(cfg.LLM.APIKeyRef); ok {
		options.APIKey = apiKey
	}
	return options, options.API != "" || options.BaseURL != "" || options.Model != "" || options.APIKey != ""
}

func applyLLMEnv(options *llm.Options) bool {
	if options == nil {
		return false
	}
	var found bool
	if value, ok := envValue(envLLMAPI); ok {
		options.API = llm.API(value)
		found = true
	}
	if value, ok := envValue(envLLMBaseURL); ok {
		options.BaseURL = value
		found = true
	}
	if value, ok := envValue(envLLMModel); ok {
		options.Model = value
		found = true
	}
	if value, ok := envValue(envLLMAPIKey); ok {
		options.APIKey = value
		found = true
	}
	return found
}

func resolveAPIKey(ref string) (string, bool) {
	ref = strings.TrimSpace(ref)
	if !strings.HasPrefix(ref, apiKeyEnvPrefix) {
		return "", false
	}
	name := strings.TrimSpace(strings.TrimPrefix(ref, apiKeyEnvPrefix))
	if name == "" {
		return "", false
	}
	return envValue(name)
}

func envValue(name string) (string, bool) {
	value := strings.TrimSpace(os.Getenv(name))
	return value, value != ""
}
