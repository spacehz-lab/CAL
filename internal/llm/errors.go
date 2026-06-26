package llm

import "errors"

// ErrEmptyResponse reports a provider response without output text.
var ErrEmptyResponse = errors.New("llm returned empty output")

// ErrMissingAPIKey reports direct provider client construction without a key.
var ErrMissingAPIKey = errors.New("llm provider api key is not configured")

// ErrMissingAPI reports partial LLM configuration without an API type.
var ErrMissingAPI = errors.New("llm api is not configured")

// ErrMissingModel reports direct provider client construction without a model.
var ErrMissingModel = errors.New("llm model is not configured")

// ErrNoClient reports a missing LLM client.
var ErrNoClient = errors.New("llm client is not configured")

// ErrUnsupportedAPI reports an unknown LLM API type.
var ErrUnsupportedAPI = errors.New("unsupported llm api")
