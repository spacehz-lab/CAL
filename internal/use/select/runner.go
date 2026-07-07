package selector

import (
	"context"
	"errors"
	"fmt"

	"github.com/spacehz-lab/cal/internal/llm"
	"github.com/spacehz-lab/cal/internal/model"
)

const (
	CodeNoMatch             = "no_match"
	CodeAmbiguous           = "ambiguous"
	CodeLLMSelectionFailed  = "llm_selection_failed"
	CodeInvalidLLMSelection = "invalid_llm_selection"
	SourceLocal             = Source("local")
	SourceLLM               = Source("llm")
)

const (
	defaultTopK = 5
)

var ErrNilRequest = errors.New("use select request is required")

// Source identifies the selector that chose a binding.
type Source string

// Error describes selection failures with stable use error codes.
type Error struct {
	Code    string
	Message string
}

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	return err.Message
}

// Request provides semantic binding selection input.
type Request struct {
	Intent         string
	Inputs         map[string]any
	ProviderID     string
	MinVerifyLevel model.VerifyLevel
	Capabilities   []model.Capability
}

// Result describes the selected promoted binding.
type Result struct {
	Source               Source
	CapabilityID         string
	BindingID            string
	ProviderID           string
	RequiredInputs       []string
	InputsPatch          map[string]any
	Reason               string
	CandidatesConsidered int
}

// Runner selects one promoted binding for an intent.
type Runner struct {
	client llm.Client
	topK   int
}

// Option customizes the selector.
type Option func(*Runner)

// NewRunner creates a semantic selector.
func NewRunner(opts ...Option) *Runner {
	runner := &Runner{topK: defaultTopK}
	for _, opt := range opts {
		opt(runner)
	}
	return runner
}

// WithLLM enables bounded LLM selection over local candidate shortlists.
func WithLLM(client llm.Client) Option {
	return func(runner *Runner) {
		runner.client = client
	}
}

// Run selects one promoted binding.
func (runner *Runner) Run(ctx context.Context, req *Request) (*Result, error) {
	if req == nil {
		return nil, ErrNilRequest
	}
	candidates := newCandidateBuilder(req).Build()
	if len(candidates) == 0 {
		return nil, &Error{Code: CodeNoMatch, Message: "no promoted binding matched the intent"}
	}
	sortCandidates(candidates)
	if runner != nil && runner.client != nil && shouldUseLLM(candidates) {
		return runner.selectWithLLM(ctx, req, candidates)
	}
	return selectLocal(candidates)
}

func selectionError(code string, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}
