# CAL Release V1 LLM

`llm/` owns the live LLM provider adapter boundary.

It is a foundation package. It turns an explicit runtime `Options` value into a
small provider-neutral client that sends one bounded prompt and returns raw
model text.

## Goal

`llm/` provides one narrow operation:

```text
system text + user text
-> OpenAI-compatible API call
-> raw response text
```

It does not know whether the caller is Proposal, Use selection, or any future
semantic helper.

## Boundary

`llm/` owns:

- Provider-neutral `Client` interface.
- Runtime-only `Options`.
- API mode constants.
- OpenAI-compatible Responses API adapter.
- OpenAI-compatible Chat Completions adapter.
- Request option construction for API key, base URL, and model.
- Empty response detection.
- Missing API, API key, model, and unsupported API errors.
- Context propagation into SDK calls.

`llm/` does not own:

- `config.json` loading.
- `CAL_HOME` resolution.
- `api_key_ref` resolution.
- Environment variable parsing.
- Prompt construction.
- Prompt wording.
- Proposal JSON parsing.
- Use selection semantics.
- Trace, run, or store writes.
- Logging.
- Retry, fallback, replay, or rules behavior.

Runtime configuration resolution belongs to the composition layer:

```text
cald/app -> config
cald/app -> env or secret lookup
cald/app -> llm.Options
cald/app -> llm.New
```

## Dependency Rule

```text
llm -> standard library
llm -> github.com/openai/openai-go/v3
```

Forbidden:

```text
llm -> config
llm -> model
llm -> store
llm -> logging
llm -> acquisition
llm -> proposal
llm -> use
llm -> contract/httpserver/cli/cald
```

`llm` must not import `config`. Durable config and runtime secret resolution are
different responsibilities.

## Files

```text
llm/
  client.go
  options.go
  openai.go
  errors.go
```

Do not add subpackages for V1.

## Public Shape

`client.go` owns the provider-neutral contract:

```go
type Client interface {
	Model() string
	Complete(context.Context, *Request) (*Response, error)
}

type Request struct {
	System string
	User   string
	JSON   bool
}

type Response struct {
	Text  string
	Model string
}
```

Use pointer parameters and pointer return values because requests and responses
are non-trivial execution values.

`JSON` is only a transport hint. It means the adapter should ask the provider
for JSON-object output when the selected API supports that. It does not define
the schema, validate JSON, or parse proposal output.

## Options

`options.go` owns runtime-only client construction settings:

```go
type API string

const (
	APIResponses       API = "responses"
	APIChatCompletions API = "chat_completions"
)

type Options struct {
	API     API
	BaseURL string
	Model   string
	APIKey  string
}

func (opts *Options) Validate() error
```

`Options` may contain secrets because it is runtime-only and must not be written
to config files, traces, logs, proposal fixtures, or normal CLI output.

`llm` should trim option fields before validation or request construction.

## Construction

`openai.go` owns construction:

```go
func New(opts *Options) (Client, error)
```

Construction rules:

- Nil options return an error.
- Empty options return an error, not a no-op client.
- Empty `API` returns `ErrMissingAPI`.
- Empty `APIKey` returns `ErrMissingAPIKey`.
- Empty `Model` returns `ErrMissingModel`.
- Unknown `API` returns `ErrUnsupportedAPI`.
- Empty `BaseURL` uses the OpenAI SDK default.
- Non-empty `BaseURL` is passed to the SDK.

`cald/app` may choose not to call `llm.New` when no LLM configuration is present.
That decision does not belong inside `llm`.

## Client Behavior

`Complete` behavior:

- Nil client returns `ErrNoClient`.
- Nil request returns an error.
- `context.Context` is passed directly to the SDK call.
- Provider API errors are returned without converting them to CAL API DTOs.
- Empty or whitespace-only provider output returns `ErrEmptyResponse`.
- Response text is trimmed before returning.
- The returned `Response.Model` is the configured model id.

Responses API behavior:

- Use `Request.System` as instructions.
- Use `Request.User` as input.
- Set `Store` to false.
- Return `response.OutputText()`.

Chat Completions behavior:

- Send one system message and one user message.
- When `Request.JSON` is true, request JSON-object response format.
- Return the first choice message content.

## Error Constants

`errors.go` owns package errors:

```go
var (
	ErrEmptyResponse  = errors.New("llm returned empty output")
	ErrMissingOptions = errors.New("llm options are required")
	ErrMissingAPI     = errors.New("llm api is not configured")
	ErrMissingAPIKey  = errors.New("llm provider api key is not configured")
	ErrMissingModel   = errors.New("llm model is not configured")
	ErrNoClient       = errors.New("llm client is not configured")
	ErrNilRequest     = errors.New("llm request is required")
	ErrUnsupportedAPI = errors.New("unsupported llm api")
)
```

Use `errors.Is`-friendly wrapping for unsupported API values.

## Testing

Unit tests should use local HTTP servers or fake OpenAI-compatible responses.

Tests must cover:

- Missing options.
- Missing API.
- Missing API key.
- Missing model.
- Unsupported API.
- Base URL forwarding.
- Responses API request shape.
- Chat Completions request shape.
- `JSON` response-format behavior for Chat Completions.
- Empty provider output.
- Context cancellation propagation where practical.

Live LLM tests do not belong in the default unit test path.
