package llm

import (
	"context"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
	"github.com/spacehz-lab/cal/internal/config"
)

// ResponsesClient executes prompts through the Responses API.
type ResponsesClient struct {
	client openai.Client
	model  string
}

// NewResponsesClient builds a Responses API client.
func NewResponsesClient(cfg config.LLMConfig, opts ...option.RequestOption) (*ResponsesClient, error) {
	requestOptions, model, err := clientOptions(cfg, opts...)
	if err != nil {
		return nil, err
	}
	return &ResponsesClient{
		client: openai.NewClient(requestOptions...),
		model:  model,
	}, nil
}

// Model returns the configured model id.
func (client *ResponsesClient) Model() string {
	if client == nil {
		return ""
	}
	return client.model
}

// Complete sends a bounded prompt and returns the raw model output.
func (client *ResponsesClient) Complete(ctx context.Context, prompt Prompt) ([]byte, error) {
	if client == nil {
		return nil, ErrNoClient
	}
	response, err := client.client.Responses.New(ctx, responses.ResponseNewParams{
		Instructions: openai.String(prompt.System),
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(prompt.User),
		},
		Model: shared.ResponsesModel(client.model),
		Store: openai.Bool(false),
	})
	if err != nil {
		return nil, err
	}
	output := strings.TrimSpace(response.OutputText())
	if output == "" {
		return nil, ErrEmptyResponse
	}
	return []byte(output), nil
}
