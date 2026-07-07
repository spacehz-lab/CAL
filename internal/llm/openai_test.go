package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
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
					"text": " {\"candidates\":[]} ",
					"annotations": []
				}]
			}]
		}`))
	}))
	defer server.Close()

	client := newTestClient(t, APIResponses, server.URL)
	response, err := client.Complete(context.Background(), &Request{
		System: "system prompt",
		User:   "user prompt",
		JSON:   true,
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if response.Text != `{"candidates":[]}` || response.Model != "test-model" {
		t.Fatalf("response = %#v, want trimmed output and model", response)
	}
	if body["model"] != "test-model" || body["instructions"] != "system prompt" || body["input"] != "user prompt" || body["store"] != false {
		t.Fatalf("body = %#v, want model, instructions, input, and disabled store", body)
	}
}

func TestChatClientCompleteCallsChatCompletionsAPIWithJSONHint(t *testing.T) {
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
					"content": " {\"candidates\":[]} "
				},
				"finish_reason": "stop"
			}]
		}`))
	}))
	defer server.Close()

	client := newTestClient(t, APIChatCompletions, server.URL)
	response, err := client.Complete(context.Background(), &Request{
		System: "system prompt",
		User:   "user prompt",
		JSON:   true,
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if response.Text != `{"candidates":[]}` || response.Model != "test-model" {
		t.Fatalf("response = %#v, want trimmed output and model", response)
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

func TestChatClientCompleteOmitsJSONFormatWithoutHint(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
				"message": {"role": "assistant", "content": "ok"},
				"finish_reason": "stop"
			}]
		}`))
	}))
	defer server.Close()

	client := newTestClient(t, APIChatCompletions, server.URL)
	if _, err := client.Complete(context.Background(), &Request{System: "system", User: "user"}); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if _, ok := body["response_format"]; ok {
		t.Fatalf("response_format = %#v, want omitted", body["response_format"])
	}
}

func TestCompleteRequiresClientRequestAndOutputText(t *testing.T) {
	var nilClient *openAIClient
	if _, err := nilClient.Complete(context.Background(), &Request{}); !errors.Is(err, ErrNoClient) {
		t.Fatalf("nil Complete() error = %v, want ErrNoClient", err)
	}

	client := newTestClient(t, APIChatCompletions, failingServer(t, `{
		"id": "chatcmpl_test",
		"object": "chat.completion",
		"created": 0,
		"model": "test-model",
		"choices": []
	}`))
	if _, err := client.Complete(context.Background(), nil); !errors.Is(err, ErrNilRequest) {
		t.Fatalf("Complete(nil) error = %v, want ErrNilRequest", err)
	}
	if _, err := client.Complete(context.Background(), &Request{}); !errors.Is(err, ErrEmptyResponse) {
		t.Fatalf("Complete() error = %v, want ErrEmptyResponse", err)
	}
}

func TestCompletePropagatesContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("server should not be called after context cancellation")
	}))
	defer server.Close()

	client := newTestClient(t, APIResponses, server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.Complete(ctx, &Request{System: "system", User: "user"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Complete() error = %v, want context.Canceled", err)
	}
}

func newTestClient(t *testing.T, api API, baseURL string) Client {
	t.Helper()
	client, err := New(&Options{
		API:     api,
		BaseURL: baseURL,
		Model:   "test-model",
		APIKey:  "test-key",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return client
}

func failingServer(t *testing.T, response string) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(server.Close)
	return server.URL
}
