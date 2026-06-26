package llm

import (
	"context"
)

// Client executes one LLM prompt.
type Client interface {
	Complete(context.Context, Prompt) ([]byte, error)
}
