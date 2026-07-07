package llm

import "errors"

var (
	// ErrEmptyResponse reports provider output without response text.
	ErrEmptyResponse = errors.New("llm returned empty output")
	// ErrMissingOptions reports nil runtime client options.
	ErrMissingOptions = errors.New("llm options are required")
	// ErrMissingAPI reports runtime client options without an API type.
	ErrMissingAPI = errors.New("llm api is not configured")
	// ErrMissingAPIKey reports runtime client options without an API key.
	ErrMissingAPIKey = errors.New("llm provider api key is not configured")
	// ErrMissingModel reports runtime client options without a model.
	ErrMissingModel = errors.New("llm model is not configured")
	// ErrNoClient reports a missing LLM client.
	ErrNoClient = errors.New("llm client is not configured")
	// ErrNilRequest reports a missing completion request.
	ErrNilRequest = errors.New("llm request is required")
	// ErrUnsupportedAPI reports an unknown API type.
	ErrUnsupportedAPI = errors.New("unsupported llm api")
)
