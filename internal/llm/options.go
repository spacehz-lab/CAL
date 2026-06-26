package llm

import (
	"strings"

	"github.com/openai/openai-go/v3/option"
	"github.com/spacehz-lab/cal/internal/config"
)

func clientOptions(cfg config.LLMConfig, opts ...option.RequestOption) ([]option.RequestOption, string, error) {
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return nil, "", ErrMissingAPIKey
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		return nil, "", ErrMissingModel
	}

	defaults := []option.RequestOption{option.WithAPIKey(apiKey)}
	if baseURL := strings.TrimSpace(cfg.BaseURL); baseURL != "" {
		defaults = append(defaults, option.WithBaseURL(baseURL))
	}
	return append(defaults, opts...), model, nil
}
