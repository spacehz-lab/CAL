package llm

import (
	"fmt"

	"github.com/spacehz-lab/cal/internal/config"
)

// NewClient builds a live LLM client from explicit runtime configuration.
func NewClient(cfg config.LLMConfig) (Client, error) {
	if cfg.Empty() {
		return nil, nil
	}
	switch cfg.API {
	case config.LLMAPIResponses:
		return NewResponsesClient(cfg)
	case config.LLMAPIChatCompletions:
		return NewChatClient(cfg)
	case "":
		return nil, ErrMissingAPI
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedAPI, cfg.API)
	}
}
