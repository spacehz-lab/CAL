package use

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/runtime"
)

const (
	// CodeInvalidInput marks malformed Use requests.
	CodeInvalidInput = "invalid_use_input"
	// CodeNoMatch marks requests where no promoted binding matches the intent.
	CodeNoMatch = "no_match"
	// CodeMissingInputs marks requests missing inputs required by a matching binding.
	CodeMissingInputs = "missing_inputs"
	// CodeAmbiguous marks requests matching multiple equally ranked capabilities.
	CodeAmbiguous = "ambiguous"
	// CodeLLMSelectionFailed marks failed LLM selection calls.
	CodeLLMSelectionFailed = "llm_selection_failed"
	// CodeInvalidLLMSelection marks LLM output that does not select a shortlisted binding.
	CodeInvalidLLMSelection = "invalid_llm_selection"
	// CodeArtifactPathFailed marks failures creating a local temporary artifact path.
	CodeArtifactPathFailed = "artifact_path_failed"

	selectionSourceLocal = "local"
)

var tokenPattern = regexp.MustCompile(`[a-z0-9]+`)

// Request describes one semantic capability use request.
type Request struct {
	Intent     string         `json:"intent"`
	Inputs     map[string]any `json:"inputs"`
	ProviderID string         `json:"provider_id,omitempty"`
	Strategy   string         `json:"strategy,omitempty"`
	Verify     bool           `json:"verify,omitempty"`
}

// Validate checks the request shape before selection.
func (req Request) Validate() *Error {
	if strings.TrimSpace(req.Intent) == "" {
		return &Error{Code: CodeInvalidInput, Message: "intent is required"}
	}
	return nil
}

// Result is the synchronous result of selecting and running a promoted binding.
type Result struct {
	ID         string            `json:"id"`
	Intent     string            `json:"intent"`
	Selection  *Selection        `json:"selection,omitempty"`
	Run        *core.Run         `json:"run,omitempty"`
	Status     core.RunStatus    `json:"status"`
	StartedAt  string            `json:"started_at,omitempty"`
	FinishedAt string            `json:"finished_at,omitempty"`
	DurationMS int64             `json:"duration_ms,omitempty"`
	Error      *core.RecordError `json:"error,omitempty"`
}

// NewResult starts a Use result record.
func NewResult(intent string, now time.Time) Result {
	return Result{
		ID:        newID(now),
		Intent:    intent,
		StartedAt: now.UTC().Format(time.RFC3339Nano),
	}
}

// Fail marks a Use result as failed.
func (result *Result) Fail(started time.Time, code, message string) {
	finished := time.Now().UTC()
	result.Status = core.RunStatusFailed
	result.FinishedAt = finished.Format(time.RFC3339Nano)
	result.DurationMS = finished.Sub(started).Milliseconds()
	result.Error = &core.RecordError{Code: code, Message: message}
}

// Complete marks a Use result as succeeded.
func (result *Result) Complete(started time.Time) {
	finished := time.Now().UTC()
	result.Status = core.RunStatusSucceeded
	result.FinishedAt = finished.Format(time.RFC3339Nano)
	result.DurationMS = finished.Sub(started).Milliseconds()
}

// Selection describes the promoted binding selected for one use request.
type Selection struct {
	Source               string `json:"source"`
	CapabilityID         string `json:"capability_id"`
	BindingID            string `json:"binding_id"`
	ProviderID           string `json:"provider_id"`
	Reason               string `json:"reason,omitempty"`
	CandidatesConsidered int    `json:"candidates_considered"`
	inputsPatch          map[string]any
}

// Error describes a Use request or selection failure.
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

// Resolver selects one promoted binding for a Use request.
type Resolver struct {
	req      Request
	runner   runtime.Runner
	selector selector
}

// ResolverOption customizes Use resolution.
type ResolverOption func(*Resolver)

// NewResolver builds a Use resolver.
func NewResolver(req Request, opts ...ResolverOption) Resolver {
	req.Inputs = normalizeInputs(req.Inputs)
	resolver := Resolver{
		req:    req,
		runner: runtime.NewRunner(runtime.DefaultRegistry()),
	}
	for _, opt := range opts {
		opt(&resolver)
	}
	return resolver
}

// Resolution is the selected binding plus its runtime input contract.
type Resolution struct {
	Selection Selection
	Required  []string
}

// Select returns the best matching promoted binding for the request.
func (resolver Resolver) Select(ctx context.Context, capabilities []core.Capability) (Selection, *Error) {
	resolution, err := resolver.Resolve(ctx, capabilities)
	if err != nil {
		return Selection{}, err
	}
	return resolution.Selection, nil
}

// Resolve returns the selected promoted binding and input contract for a request.
func (resolver Resolver) Resolve(ctx context.Context, capabilities []core.Capability) (Resolution, *Error) {
	intentTokens := tokenizeText(resolver.req.Intent)
	var candidates []candidate
	considered := 0

	for _, capability := range capabilities {
		for _, binding := range capability.Bindings {
			candidate, ok := resolver.scoreBinding(capability, binding, intentTokens)
			if !ok {
				continue
			}
			considered++
			candidates = append(candidates, candidate)
		}
	}
	if len(candidates) == 0 {
		return Resolution{}, &Error{Code: CodeNoMatch, Message: "no promoted binding matched the intent"}
	}

	sortCandidates(candidates)
	selection, err := resolver.selectBinding(ctx, candidates)
	if err != nil {
		return Resolution{}, err
	}
	selection.CandidatesConsidered = considered
	selected, ok := findCandidate(candidates, selection.BindingID)
	if !ok {
		return Resolution{}, &Error{Code: CodeInvalidLLMSelection, Message: fmt.Sprintf("selected unknown binding %q", selection.BindingID)}
	}
	return Resolution{
		Selection: selection,
		Required:  selected.required,
	}, nil
}

type candidate struct {
	capability core.Capability
	binding    core.Binding
	required   []string
	missing    []string
	score      int
}

func (resolver Resolver) scoreBinding(capability core.Capability, binding core.Binding, intentTokens map[string]struct{}) (candidate, bool) {
	if binding.State != core.BindingStatePromoted {
		return candidate{}, false
	}
	if resolver.req.ProviderID != "" && binding.ProviderID != resolver.req.ProviderID {
		return candidate{}, false
	}
	if resolver.req.Verify && binding.Verifier == nil {
		return candidate{}, false
	}
	if !resolver.runner.Supports(binding.Execution.Kind) {
		return candidate{}, false
	}
	required, err := resolver.runner.RequiredInputs(binding.Execution)
	if err != nil {
		return candidate{}, false
	}

	score := intentScore(capability, intentTokens)
	if score == 0 {
		return candidate{}, false
	}
	missing := missingInputs(required, resolver.req.Inputs)
	score += len(required) - len(missing)
	if resolver.req.ProviderID != "" {
		score += 4
	}
	return candidate{
		capability: capability,
		binding:    binding,
		required:   required,
		missing:    missing,
		score:      score,
	}, true
}

func intentScore(capability core.Capability, intentTokens map[string]struct{}) int {
	score := 0
	for token := range tokenizeText(capability.ID) {
		if _, ok := intentTokens[token]; ok {
			score += 4
		}
	}
	for token := range tokenizeText(capability.Description) {
		if _, ok := intentTokens[token]; ok {
			score += 2
		}
	}
	return score
}

func tokenizeText(text string) map[string]struct{} {
	tokens := map[string]struct{}{}
	text = strings.ToLower(text)
	for _, token := range tokenPattern.FindAllString(text, -1) {
		tokens[token] = struct{}{}
	}
	return tokens
}

func missingInputs(required []string, inputs map[string]any) []string {
	var missing []string
	for _, name := range required {
		value, ok := inputs[name]
		if !ok || value == nil || value == "" {
			missing = append(missing, name)
		}
	}
	return missing
}

func normalizeInputs(inputs map[string]any) map[string]any {
	if inputs == nil {
		return map[string]any{}
	}
	return inputs
}

func findCandidate(candidates []candidate, bindingID string) (candidate, bool) {
	for _, candidate := range candidates {
		if candidate.binding.ID == bindingID {
			return candidate, true
		}
	}
	return candidate{}, false
}

func hasMissingNonTargetInputs(candidates []candidate) bool {
	for _, candidate := range candidates {
		for _, name := range candidate.missing {
			if name != "target" {
				return true
			}
		}
	}
	return false
}

func sortCandidates(candidates []candidate) {
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		if candidates[i].capability.ID != candidates[j].capability.ID {
			return candidates[i].capability.ID < candidates[j].capability.ID
		}
		return candidates[i].binding.ID < candidates[j].binding.ID
	})
}

func (resolver Resolver) selectBinding(ctx context.Context, candidates []candidate) (Selection, *Error) {
	if resolver.selector != nil && (len(candidates) > 1 || hasMissingNonTargetInputs(candidates)) {
		return resolver.selector.selectBinding(ctx, resolver.req, candidates)
	}
	return localSelector{}.selectBinding(ctx, resolver.req, candidates)
}

func newID(now time.Time) string {
	return "use_" + fmt.Sprint(now.UTC().UnixNano())
}
