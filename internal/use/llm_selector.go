package use

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spacehz-lab/cal/internal/core"
	sharedllm "github.com/spacehz-lab/cal/internal/llm"
)

const (
	selectionSourceLLM = "llm"
	llmTopK            = 5
)

// WithLLM enables bounded LLM selection over the local candidate shortlist.
func WithLLM(client sharedllm.Client) ResolverOption {
	return func(resolver *Resolver) {
		if client != nil {
			resolver.selector = llmSelector{client: client}
		}
	}
}

type llmSelector struct {
	client sharedllm.Client
}

func (selector llmSelector) selectBinding(ctx context.Context, req Request, candidates []candidate) (Selection, *Error) {
	if selector.client == nil {
		return localSelector{}.selectBinding(ctx, req, candidates)
	}
	shortlist := candidates
	if len(shortlist) > llmTopK {
		shortlist = shortlist[:llmTopK]
	}
	prompt, bindingByID, err := buildLLMPrompt(req, shortlist)
	if err != nil {
		return Selection{}, &Error{Code: CodeLLMSelectionFailed, Message: err.Error()}
	}
	content, callErr := selector.client.Complete(ctx, prompt)
	if callErr != nil {
		return Selection{}, &Error{Code: CodeLLMSelectionFailed, Message: callErr.Error()}
	}
	var decision llmDecision
	if err := json.Unmarshal(content, &decision); err != nil {
		return Selection{}, &Error{Code: CodeInvalidLLMSelection, Message: fmt.Sprintf("decode llm selection: %v", err)}
	}
	bindingID := strings.TrimSpace(decision.BindingID)
	selected, ok := bindingByID[bindingID]
	if !ok {
		return Selection{}, &Error{Code: CodeInvalidLLMSelection, Message: fmt.Sprintf("llm selected unknown binding %q", bindingID)}
	}
	inputsPatch, patchErr := validateInputsPatch(req, selected, decision.InputsPatch)
	if patchErr != nil {
		return Selection{}, patchErr
	}
	reason := strings.TrimSpace(decision.Reason)
	if reason == "" {
		reason = "llm selected this promoted binding from the local shortlist"
	}
	return Selection{
		Source:       selectionSourceLLM,
		CapabilityID: selected.capability.ID,
		BindingID:    selected.binding.ID,
		ProviderID:   selected.binding.ProviderID,
		Reason:       reason,
		inputsPatch:  inputsPatch,
	}, nil
}

type llmDecision struct {
	BindingID   string         `json:"binding_id"`
	InputsPatch map[string]any `json:"inputs_patch,omitempty"`
	Reason      string         `json:"reason,omitempty"`
}

type llmRequest struct {
	Intent     string    `json:"intent"`
	InputKeys  []string  `json:"input_keys"`
	Candidates []llmCard `json:"candidates"`
}

type llmCard struct {
	CapabilityID          string         `json:"capability_id"`
	CapabilityDescription string         `json:"capability_description,omitempty"`
	BindingID             string         `json:"binding_id"`
	ProviderID            string         `json:"provider_id"`
	ExecutionKind         string         `json:"execution_kind"`
	RequiredInputs        []string       `json:"required_inputs,omitempty"`
	InputConstraints      map[string]any `json:"input_constraints,omitempty"`
	ExecutionArgs         []string       `json:"execution_args,omitempty"`
}

func buildLLMPrompt(req Request, candidates []candidate) (sharedllm.Prompt, map[string]candidate, error) {
	bindingByID := make(map[string]candidate, len(candidates))
	cards := make([]llmCard, 0, len(candidates))
	for _, candidate := range candidates {
		bindingByID[candidate.binding.ID] = candidate
		cards = append(cards, llmCard{
			CapabilityID:          candidate.capability.ID,
			CapabilityDescription: candidate.capability.Description,
			BindingID:             candidate.binding.ID,
			ProviderID:            candidate.binding.ProviderID,
			ExecutionKind:         string(candidate.binding.Execution.Kind),
			RequiredInputs:        candidate.required,
			InputConstraints:      candidate.binding.InputConstraints,
			ExecutionArgs:         executionArgs(candidate.binding.Execution),
		})
	}
	payload := llmRequest{
		Intent:     req.Intent,
		InputKeys:  inputKeys(req.Inputs),
		Candidates: cards,
	}
	content, err := json.Marshal(payload)
	if err != nil {
		return sharedllm.Prompt{}, nil, err
	}
	return sharedllm.Prompt{
		System: llmSystemPrompt,
		User:   string(content),
	}, bindingByID, nil
}

const llmSystemPrompt = `You select one already-promoted CAL binding for a user intent.
Return JSON only: {"binding_id":"...","inputs_patch":{},"reason":"short reason"}.
Choose only from candidates[].binding_id.
inputs_patch may include only values explicitly present in the intent and required by the selected candidate.
Do not include target in inputs_patch; CAL creates missing target paths locally.
Do not overwrite input_keys already supplied by the caller.
Do not invent capabilities, bindings, inputs, commands, files, or outcomes.
Do not claim execution or verification success; CAL validates and runs after selection.`

func validateInputsPatch(req Request, selected candidate, patch map[string]any) (map[string]any, *Error) {
	if len(patch) == 0 {
		return nil, nil
	}
	allowed := map[string]struct{}{}
	for _, name := range selected.required {
		allowed[name] = struct{}{}
	}
	cleaned := make(map[string]any, len(patch))
	for name, value := range patch {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, &Error{Code: CodeInvalidLLMSelection, Message: "llm inputs_patch contains an empty key"}
		}
		if name == "target" {
			return nil, &Error{Code: CodeInvalidLLMSelection, Message: "llm inputs_patch must not include target"}
		}
		if _, ok := allowed[name]; !ok {
			return nil, &Error{Code: CodeInvalidLLMSelection, Message: fmt.Sprintf("llm inputs_patch key %q is not required by selected binding", name)}
		}
		if hasInput(req.Inputs, name) {
			return nil, &Error{Code: CodeInvalidLLMSelection, Message: fmt.Sprintf("llm inputs_patch key %q overwrites caller input", name)}
		}
		cleaned[name] = value
	}
	return cleaned, nil
}

func inputKeys(inputs map[string]any) []string {
	keys := make([]string, 0, len(inputs))
	for key := range inputs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func executionArgs(execution core.Execution) []string {
	value, ok := execution.Spec[core.ExecutionSpecArgs]
	if !ok {
		return nil
	}
	switch args := value.(type) {
	case []string:
		return append([]string(nil), args...)
	case []any:
		out := make([]string, 0, len(args))
		for _, arg := range args {
			text, ok := arg.(string)
			if !ok {
				continue
			}
			out = append(out, text)
		}
		return out
	default:
		return nil
	}
}
