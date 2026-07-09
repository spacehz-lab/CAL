package llm

import "context"

// Client executes one bounded LLM request.
type Client interface {
	Model() string
	Complete(context.Context, *Request) (*Response, error)
}

// Request is the provider-neutral shape for one LLM call.
type Request struct {
	System string
	User   string
	JSON   bool
}

// Response is raw model output plus local provenance.
type Response struct {
	Text  string
	Model string
	Usage Usage
}

// Usage reports provider token accounting when the provider returns it.
type Usage struct {
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
}
