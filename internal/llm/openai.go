package llm

import (
	"context"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
)

type openAIClient struct {
	api    API
	client openai.Client
	model  string
}

// New builds an OpenAI-compatible LLM client from runtime options.
func New(opts *Options) (Client, error) {
	cleaned, err := cleanOptions(opts)
	if err != nil {
		return nil, err
	}
	return newOpenAIClient(cleaned), nil
}

// Model returns the configured model id.
func (client *openAIClient) Model() string {
	if client == nil {
		return ""
	}
	return client.model
}

// Complete sends one request and returns trimmed raw model text.
func (client *openAIClient) Complete(ctx context.Context, req *Request) (*Response, error) {
	if client == nil {
		return nil, ErrNoClient
	}
	if req == nil {
		return nil, ErrNilRequest
	}
	switch client.api {
	case APIResponses:
		return client.completeResponses(ctx, req)
	case APIChatCompletions:
		return client.completeChat(ctx, req)
	default:
		return nil, ErrUnsupportedAPI
	}
}

func (client *openAIClient) completeResponses(ctx context.Context, req *Request) (*Response, error) {
	response, err := client.client.Responses.New(ctx, responses.ResponseNewParams{
		Instructions: openai.String(req.System),
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(req.User),
		},
		Model: shared.ResponsesModel(client.model),
		Store: openai.Bool(false),
	})
	if err != nil {
		return nil, err
	}
	return client.response(response.OutputText(), Usage{
		PromptTokens:     response.Usage.InputTokens,
		CompletionTokens: response.Usage.OutputTokens,
		TotalTokens:      response.Usage.TotalTokens,
	})
}

func (client *openAIClient) completeChat(ctx context.Context, req *Request) (*Response, error) {
	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(req.System),
			openai.UserMessage(req.User),
		},
		Model: shared.ChatModel(client.model),
	}
	if req.JSON {
		params.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &shared.ResponseFormatJSONObjectParam{},
		}
	}

	completion, err := client.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, err
	}
	if len(completion.Choices) == 0 {
		return nil, ErrEmptyResponse
	}
	return client.response(completion.Choices[0].Message.Content, Usage{
		PromptTokens:     completion.Usage.PromptTokens,
		CompletionTokens: completion.Usage.CompletionTokens,
		TotalTokens:      completion.Usage.TotalTokens,
	})
}

func (client *openAIClient) response(text string, usage Usage) (*Response, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrEmptyResponse
	}
	return &Response{Text: text, Model: client.model, Usage: usage}, nil
}

func newOpenAIClient(opts Options) *openAIClient {
	requestOptions := []option.RequestOption{option.WithAPIKey(opts.APIKey)}
	if opts.BaseURL != "" {
		requestOptions = append(requestOptions, option.WithBaseURL(opts.BaseURL))
	}
	return &openAIClient{
		api:    opts.API,
		client: openai.NewClient(requestOptions...),
		model:  opts.Model,
	}
}
