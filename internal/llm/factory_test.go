package llm

import (
	"errors"
	"testing"

	"github.com/spacehz-lab/cal/internal/config"
)

func TestNewClientSelectsConfiguredAPI(t *testing.T) {
	base := config.LLMConfig{APIKey: "test-key", Model: "test-model"}

	client, err := NewClient(config.LLMConfig{
		API:    config.LLMAPIResponses,
		APIKey: base.APIKey,
		Model:  base.Model,
	})
	if err != nil {
		t.Fatalf("NewClient(responses) error = %v", err)
	}
	if _, ok := client.(*ResponsesClient); !ok {
		t.Fatalf("NewClient(responses) = %T, want *ResponsesClient", client)
	}

	client, err = NewClient(config.LLMConfig{
		API:    config.LLMAPIChatCompletions,
		APIKey: base.APIKey,
		Model:  base.Model,
	})
	if err != nil {
		t.Fatalf("NewClient(chat_completions) error = %v", err)
	}
	if _, ok := client.(*ChatClient); !ok {
		t.Fatalf("NewClient(chat_completions) = %T, want *ChatClient", client)
	}
}

func TestNewClientDisabledWhenConfigEmpty(t *testing.T) {
	client, err := NewClient(config.LLMConfig{})
	if err != nil || client != nil {
		t.Fatalf("NewClient(empty) = (%#v, %v), want nil client and nil error", client, err)
	}
}

func TestNewClientRejectsPartialOrUnknownConfig(t *testing.T) {
	if _, err := NewClient(config.LLMConfig{APIKey: "test-key"}); !errors.Is(err, ErrMissingAPI) {
		t.Fatalf("NewClient(partial) error = %v, want ErrMissingAPI", err)
	}
	if _, err := NewClient(config.LLMConfig{API: "unknown", APIKey: "test-key", Model: "test-model"}); !errors.Is(err, ErrUnsupportedAPI) {
		t.Fatalf("NewClient(unknown) error = %v, want ErrUnsupportedAPI", err)
	}
}
