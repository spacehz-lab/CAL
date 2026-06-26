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

func TestResponsesClientCompleteCallsResponsesAPI(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			t.Fatalf("path = %q, want /responses", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want bearer test key", auth)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "resp_test",
			"object": "response",
			"created_at": 0,
			"status": "completed",
			"model": "test-model",
			"output": [{
				"id": "msg_test",
				"type": "message",
				"status": "completed",
				"role": "assistant",
				"content": [{
					"type": "output_text",
					"text": "{\"candidates\":[]}",
					"annotations": []
				}]
			}]
		}`))
	}))
	defer server.Close()

	client, err := NewResponsesClient(testLLMConfig(server.URL), option.WithMaxRetries(0))
	if err != nil {
		t.Fatalf("NewResponsesClient() error = %v", err)
	}
	output, err := client.Complete(context.Background(), Prompt{
		System: "system prompt",
		User:   "user prompt",
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if string(output) != `{"candidates":[]}` {
		t.Fatalf("output = %q, want response output text", output)
	}
	if body["model"] != "test-model" || body["instructions"] != "system prompt" || body["input"] != "user prompt" || body["store"] != false {
		t.Fatalf("body = %#v, want model, instructions, input, and disabled store", body)
	}
}

func TestResponsesClientCompleteRequiresOutputText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "resp_test",
			"object": "response",
			"created_at": 0,
			"status": "completed",
			"model": "test-model",
			"output": []
		}`))
	}))
	defer server.Close()

	client, err := NewResponsesClient(testLLMConfig(server.URL), option.WithMaxRetries(0))
	if err != nil {
		t.Fatalf("NewResponsesClient() error = %v", err)
	}
	_, err = client.Complete(context.Background(), Prompt{})
	if !errors.Is(err, ErrEmptyResponse) {
		t.Fatalf("Complete() error = %v, want ErrEmptyResponse", err)
	}
}

func TestNewResponsesClientRequiresExplicitConfig(t *testing.T) {
	if _, err := NewResponsesClient(config.LLMConfig{Model: "test-model"}); !errors.Is(err, ErrMissingAPIKey) {
		t.Fatalf("NewResponsesClient() error = %v, want ErrMissingAPIKey", err)
	}
	if _, err := NewResponsesClient(config.LLMConfig{APIKey: "test-key"}); !errors.Is(err, ErrMissingModel) {
		t.Fatalf("NewResponsesClient() error = %v, want ErrMissingModel", err)
	}
}

func testLLMConfig(baseURL string) config.LLMConfig {
	return config.LLMConfig{
		BaseURL: baseURL,
		Model:   "test-model",
		APIKey:  "test-key",
	}
}
