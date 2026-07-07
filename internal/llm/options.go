package llm

import (
	"fmt"
	"strings"
)

// API identifies the OpenAI-compatible API mode.
type API string

const (
	// APIResponses selects the OpenAI Responses API.
	APIResponses API = "responses"
	// APIChatCompletions selects an OpenAI-compatible Chat Completions API.
	APIChatCompletions API = "chat_completions"
)

// Options contains runtime-only LLM client settings.
type Options struct {
	API     API
	BaseURL string
	Model   string
	APIKey  string
}

// Validate checks runtime LLM client settings.
func (opts *Options) Validate() error {
	cleaned, err := cleanOptions(opts)
	if err != nil {
		return err
	}
	*opts = cleaned
	return nil
}

func cleanOptions(opts *Options) (Options, error) {
	if opts == nil {
		return Options{}, ErrMissingOptions
	}
	cleaned := Options{
		API:     API(strings.TrimSpace(string(opts.API))),
		BaseURL: strings.TrimSpace(opts.BaseURL),
		Model:   strings.TrimSpace(opts.Model),
		APIKey:  strings.TrimSpace(opts.APIKey),
	}
	if cleaned.API == "" {
		return Options{}, ErrMissingAPI
	}
	if cleaned.APIKey == "" {
		return Options{}, ErrMissingAPIKey
	}
	if cleaned.Model == "" {
		return Options{}, ErrMissingModel
	}
	switch cleaned.API {
	case APIResponses, APIChatCompletions:
		return cleaned, nil
	default:
		return Options{}, fmt.Errorf("%w: %s", ErrUnsupportedAPI, cleaned.API)
	}
}
