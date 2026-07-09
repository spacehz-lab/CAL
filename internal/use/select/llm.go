package selector

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spacehz-lab/cal/internal/llm"
	"github.com/spacehz-lab/cal/internal/model"
)

const (
	inputSummaryKindKey     = "kind"
	inputSummaryBasenameKey = "basename"
	inputSummaryPathKind    = "path"
)

const llmSystemPrompt = `You select one already-promoted CAL binding for a user intent.

Choose only from candidates[].binding_id.

Choose the binding that best matches the intent and can be satisfied by caller inputs plus a valid inputs_patch.
Prefer candidates with fewer missing non-target inputs.
Prefer explicit intent matches over inferred matches.
Use provider_command and execution_args to distinguish operation direction, output format, and required inputs.

Return exactly one JSON object:
{"binding_id":"...","inputs_patch":{},"reason":"short reason"}

Rules:
- binding_id must be one of candidates[].binding_id.
- inputs_patch may include only keys required by the selected candidate.
- inputs_patch may copy or rename values from caller inputs when required by the selected candidate.
Do not include target in inputs_patch; CAL creates missing target paths locally.
Do not overwrite input_keys already supplied by the caller.
- Do not invent values that are not present in the user intent or caller inputs.
- Do not invent capabilities, bindings, inputs, commands, files, paths, formats, algorithms, or outcomes.
- Do not claim execution or verification success; CAL validates and runs after selection.
- Do not include markdown, comments, or extra text.`

func (runner *Runner) selectWithLLM(ctx context.Context, req *Request, candidates []candidate) (*Result, error) {
	shortlist := candidates
	if len(shortlist) > runner.topK {
		shortlist = shortlist[:runner.topK]
	}
	request, bindingByID, err := buildLLMRequest(req, shortlist)
	if err != nil {
		return nil, &Error{Code: CodeLLMSelectionFailed, Message: err.Error()}
	}
	response, err := runner.client.Complete(ctx, request)
	if err != nil {
		return nil, &Error{Code: CodeLLMSelectionFailed, Message: err.Error()}
	}
	if response == nil {
		return nil, &Error{Code: CodeLLMSelectionFailed, Message: "llm selection returned empty response"}
	}
	var decision llmDecision
	if err := json.Unmarshal([]byte(response.Text), &decision); err != nil {
		return nil, selectionError(CodeInvalidLLMSelection, "decode llm selection: %v", err)
	}
	bindingID := strings.TrimSpace(decision.BindingID)
	selected, ok := bindingByID[bindingID]
	if !ok {
		return nil, selectionError(CodeInvalidLLMSelection, "llm selected unknown binding %q", bindingID)
	}
	inputsPatch, patchErr := validateInputsPatch(req, selected, decision.InputsPatch)
	if patchErr != nil {
		return nil, patchErr
	}
	reason := strings.TrimSpace(decision.Reason)
	if reason == "" {
		reason = "llm selected this promoted binding from the local shortlist"
	}
	return resultFromCandidate(SourceLLM, selected, reason, len(candidates), inputsPatch), nil
}

type llmDecision struct {
	BindingID   string         `json:"binding_id"`
	InputsPatch map[string]any `json:"inputs_patch,omitempty"`
	Reason      string         `json:"reason,omitempty"`
}

type llmPayload struct {
	Intent     string         `json:"intent"`
	InputKeys  []string       `json:"input_keys"`
	Inputs     map[string]any `json:"inputs,omitempty"`
	Candidates []llmCard      `json:"candidates"`
}

type llmCard struct {
	CapabilityID          string   `json:"capability_id"`
	CapabilityDescription string   `json:"capability_description,omitempty"`
	BindingID             string   `json:"binding_id"`
	ProviderID            string   `json:"provider_id"`
	ProviderCommand       string   `json:"provider_command,omitempty"`
	ExecutionKind         string   `json:"execution_kind"`
	RequiredInputs        []string `json:"required_inputs,omitempty"`
	ExecutionArgs         []string `json:"execution_args,omitempty"`
}

func buildLLMRequest(req *Request, candidates []candidate) (*llm.Request, map[string]candidate, error) {
	bindingByID := make(map[string]candidate, len(candidates))
	cards := make([]llmCard, 0, len(candidates))
	for _, candidate := range candidates {
		bindingByID[candidate.binding.ID] = candidate
		cards = append(cards, llmCard{
			CapabilityID:          candidate.capability.ID,
			CapabilityDescription: candidate.capability.Description,
			BindingID:             candidate.binding.ID,
			ProviderID:            candidate.binding.ProviderID,
			ProviderCommand:       providerCommand(req, candidate.binding.ProviderID),
			ExecutionKind:         string(candidate.binding.Execution.Kind),
			RequiredInputs:        candidate.required,
			ExecutionArgs:         executionArgs(candidate.binding.Execution),
		})
	}
	payload := llmPayload{
		Intent:     req.Intent,
		InputKeys:  inputKeys(req.Inputs),
		Inputs:     inputSummaries(req.Inputs),
		Candidates: cards,
	}
	content, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}
	return &llm.Request{System: llmSystemPrompt, User: string(content), JSON: true}, bindingByID, nil
}

func validateInputsPatch(req *Request, selected candidate, patch map[string]any) (map[string]any, *Error) {
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
		if name == targetInputName {
			return nil, &Error{Code: CodeInvalidLLMSelection, Message: "llm inputs_patch must not include target"}
		}
		if _, ok := allowed[name]; !ok {
			return nil, selectionError(CodeInvalidLLMSelection, "llm inputs_patch key %q is not required by selected binding", name)
		}
		if hasInput(req.Inputs, name) {
			return nil, selectionError(CodeInvalidLLMSelection, "llm inputs_patch key %q overwrites caller input", name)
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

func inputSummaries(inputs map[string]any) map[string]any {
	if len(inputs) == 0 {
		return nil
	}
	summaries := make(map[string]any, len(inputs))
	for key, value := range inputs {
		summaries[key] = inputSummary(value)
	}
	return summaries
}

func inputSummary(value any) any {
	switch typed := value.(type) {
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return ""
		}
		if looksLikePath(text) {
			return map[string]string{inputSummaryKindKey: inputSummaryPathKind, inputSummaryBasenameKey: filepath.Base(text)}
		}
		return text
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

func providerCommand(req *Request, providerID string) string {
	if req == nil || len(req.ProviderCommands) == 0 {
		return ""
	}
	return strings.TrimSpace(req.ProviderCommands[providerID])
}

func executionArgs(execution model.Execution) []string {
	value, ok := execution.Spec[model.ExecutionSpecArgs]
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
			if ok {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func looksLikePath(text string) bool {
	if strings.ContainsAny(text, `/\`) {
		return true
	}
	return filepath.Base(text) != text
}

func hasInput(inputs map[string]any, name string) bool {
	value, ok := inputs[name]
	return ok && value != nil && value != ""
}
