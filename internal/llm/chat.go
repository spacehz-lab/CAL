package llm

import (
	"context"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
	"github.com/spacehz-lab/cal/internal/config"
)

// ChatClient executes prompts through Chat Completions.
type ChatClient struct {
	client openai.Client
	model  string
}

// NewChatClient builds an OpenAI-compatible Chat Completions client.
func NewChatClient(cfg config.LLMConfig, opts ...option.RequestOption) (*ChatClient, error) {
	requestOptions, model, err := clientOptions(cfg, opts...)
	if err != nil {
		return nil, err
	}
	return &ChatClient{
		client: openai.NewClient(requestOptions...),
		model:  model,
	}, nil
}

// Model returns the configured model id.
func (client *ChatClient) Model() string {
	if client == nil {
		return ""
	}
	return client.model
}

// Complete sends a bounded prompt and returns the raw model output.
func (client *ChatClient) Complete(ctx context.Context, prompt Prompt) ([]byte, error) {
	if client == nil {
		return nil, ErrNoClient
	}
	completion, err := client.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(prompt.System),
			openai.UserMessage(prompt.User),
		},
		Model: shared.ChatModel(client.model),
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &shared.ResponseFormatJSONObjectParam{},
		},
	})
	if err != nil {
		return nil, err
	}
	if len(completion.Choices) == 0 {
		return nil, ErrEmptyResponse
	}
	output := strings.TrimSpace(completion.Choices[0].Message.Content)
	if output == "" {
		return nil, ErrEmptyResponse
	}
	return []byte(output), nil
}
