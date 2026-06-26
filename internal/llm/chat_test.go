package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openai/openai-go/v3/option"
	"github.com/spacehz-lab/cal/internal/config"
)

func TestChatClientCompleteCallsChatCompletionsAPI(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %q, want /chat/completions", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want bearer test key", auth)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "chatcmpl_test",
			"object": "chat.completion",
			"created": 0,
			"model": "test-model",
			"choices": [{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "{\"candidates\":[]}"
				},
				"finish_reason": "stop"
			}]
		}`))
	}))
	defer server.Close()

	client, err := NewChatClient(testLLMConfig(server.URL), option.WithMaxRetries(0))
	if err != nil {
		t.Fatalf("NewChatClient() error = %v", err)
	}
	output, err := client.Complete(context.Background(), Prompt{
		System: "system prompt",
		User:   "user prompt",
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if string(output) != `{"candidates":[]}` {
		t.Fatalf("output = %q, want chat completion message content", output)
	}
	messages, ok := body["messages"].([]any)
	if !ok || len(messages) != 2 {
		t.Fatalf("messages = %#v, want system and user messages", body["messages"])
	}
	if body["model"] != "test-model" {
		t.Fatalf("body = %#v, want model", body)
	}
	responseFormat, ok := body["response_format"].(map[string]any)
	if !ok || responseFormat["type"] != "json_object" {
		t.Fatalf("response_format = %#v, want json_object", body["response_format"])
	}
}

func TestChatClientCompleteRequiresChoiceText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "chatcmpl_test",
			"object": "chat.completion",
			"created": 0,
			"model": "test-model",
			"choices": []
		}`))
	}))
	defer server.Close()

	client, err := NewChatClient(testLLMConfig(server.URL), option.WithMaxRetries(0))
	if err != nil {
		t.Fatalf("NewChatClient() error = %v", err)
	}
	_, err = client.Complete(context.Background(), Prompt{})
	if !errors.Is(err, ErrEmptyResponse) {
		t.Fatalf("Complete() error = %v, want ErrEmptyResponse", err)
	}
}

func TestNewChatClientRequiresExplicitConfig(t *testing.T) {
	if _, err := NewChatClient(config.LLMConfig{Model: "test-model"}); !errors.Is(err, ErrMissingAPIKey) {
		t.Fatalf("NewChatClient() error = %v, want ErrMissingAPIKey", err)
	}
	if _, err := NewChatClient(config.LLMConfig{APIKey: "test-key"}); !errors.Is(err, ErrMissingModel) {
		t.Fatalf("NewChatClient() error = %v, want ErrMissingModel", err)
	}
}
