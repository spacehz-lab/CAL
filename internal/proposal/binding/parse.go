package binding

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spacehz-lab/cal/internal/model"
)

type output struct {
	Candidates    []model.Candidate `json:"candidates"`
	ProbeMaterial []ProbeMaterial   `json:"probe_material"`
}

const (
	skipReasonMissingProbeMaterial = "missing probe material"
	placeholderProviderIDOptional  = "optional"
	placeholderProviderIDSame      = "same as request provider"
)

// Parse decodes and locally filters binding-stage JSON.
func Parse(raw string, req *Request) ([]model.Candidate, []ProbeMaterial, model.ProposalStage, error) {
	var out output
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, nil, model.ProposalStage{}, fmt.Errorf("decode binding stage: %w", err)
	}
	candidates, materials, stage := normalize(out.Candidates, out.ProbeMaterial, req)
	if len(candidates) == 0 {
		if len(out.Candidates) > 0 && stageHasSkipReason(stage, skipReasonMissingProbeMaterial) {
			return nil, nil, stage, fmt.Errorf("binding stage returned candidates without matching probe material")
		}
		return nil, nil, stage, fmt.Errorf("binding stage returned no candidates")
	}
	if len(materials) != len(candidates) {
		return nil, nil, stage, fmt.Errorf("binding stage returned %d candidates and %d probe material records", len(candidates), len(materials))
	}
	return candidates, materials, stage, nil
}

func normalize(input []model.Candidate, materialInput []ProbeMaterial, req *Request) ([]model.Candidate, []ProbeMaterial, model.ProposalStage) {
	if req == nil {
		req = &Request{}
	}
	materialsByRawIndex := materialIndex(materialInput)
	stage := newStage(model.ProposalStageBinding, len(input))
	seen := map[string]struct{}{}
	candidates := make([]model.Candidate, 0, len(input))
	materials := make([]ProbeMaterial, 0, len(input))
	for index, candidate := range input {
		candidate = normalizeCandidate(candidate, req)
		trace := traceItem(index, candidate)
		material, hasMaterial := materialsByRawIndex[index]
		key, keyErr := model.CanonicalExecution(candidate.Execution)
		switch {
		case candidate.ProviderID == "":
			trace.Decision = model.ProposalDecisionSkip
			trace.Reason = "missing provider id"
		case candidate.CapabilityID == "":
			trace.Decision = model.ProposalDecisionSkip
			trace.Reason = "missing capability id"
		case candidate.Execution.Kind == "":
			trace.Decision = model.ProposalDecisionSkip
			trace.Reason = "missing execution kind"
		case keyErr != nil:
			trace.Decision = model.ProposalDecisionSkip
			trace.Reason = keyErr.Error()
		case !hasMaterial:
			trace.Decision = model.ProposalDecisionSkip
			trace.Reason = skipReasonMissingProbeMaterial
		default:
			identity := candidate.ProviderID + "|" + candidate.CapabilityID + "|" + key
			if _, ok := seen[identity]; ok {
				trace.Decision = model.ProposalDecisionSkip
				trace.Reason = "duplicate candidate"
			} else if req.MaxCandidates > 0 && len(candidates) >= req.MaxCandidates {
				trace.Decision = model.ProposalDecisionDefer
			} else {
				seen[identity] = struct{}{}
				trace.Decision = model.ProposalDecisionKeep
				material.CandidateIndex = len(candidates)
				candidates = append(candidates, candidate)
				materials = append(materials, normalizeMaterial(material))
			}
		}
		stage.Items = append(stage.Items, trace)
		stage.Summary[summaryKey(trace.Decision)]++
	}
	stage.Summary[model.ProposalSummarySelected] = len(candidates)
	return candidates, materials, stage
}

func normalizeCandidate(candidate model.Candidate, req *Request) model.Candidate {
	candidate.ProviderID = normalizeProviderID(candidate.ProviderID)
	if candidate.ProviderID == "" && req.Provider != nil {
		candidate.ProviderID = req.Provider.ID
	}
	candidate.CapabilityID = strings.TrimSpace(candidate.CapabilityID)
	if candidate.CapabilityID == "" {
		candidate.CapabilityID = req.Capability.CapabilityID
	}
	candidate.Description = strings.TrimSpace(candidate.Description)
	if candidate.Description == "" {
		candidate.Description = req.Capability.Description
	}
	candidate.Source = strings.TrimSpace(candidate.Source)
	candidate.Execution.Kind = model.ExecutionKind(strings.TrimSpace(string(candidate.Execution.Kind)))
	return candidate
}

func normalizeProviderID(providerID string) string {
	providerID = strings.TrimSpace(providerID)
	switch strings.ToLower(providerID) {
	case placeholderProviderIDOptional, placeholderProviderIDSame:
		return ""
	default:
		return providerID
	}
}

func materialIndex(materials []ProbeMaterial) map[int]ProbeMaterial {
	result := make(map[int]ProbeMaterial, len(materials))
	for _, material := range materials {
		if material.CandidateIndex < 0 {
			continue
		}
		if _, ok := result[material.CandidateIndex]; ok {
			continue
		}
		result[material.CandidateIndex] = material
	}
	return result
}

func normalizeMaterial(material ProbeMaterial) ProbeMaterial {
	for index := range material.Fixtures {
		material.Fixtures[index].Input = strings.TrimSpace(material.Fixtures[index].Input)
		material.Fixtures[index].Filename = strings.TrimSpace(material.Fixtures[index].Filename)
	}
	if material.Inputs == nil {
		material.Inputs = map[string]any{}
	}
	return material
}

func stageHasSkipReason(stage model.ProposalStage, reason string) bool {
	for _, item := range stage.Items {
		if item.Decision == model.ProposalDecisionSkip && item.Reason == reason {
			return true
		}
	}
	return false
}

func newStage(name model.ProposalStageName, raw int) model.ProposalStage {
	return model.ProposalStage{Name: name, Summary: map[model.ProposalSummaryKey]int{model.ProposalSummaryRaw: raw}}
}

func summaryKey(decision model.ProposalDecision) model.ProposalSummaryKey {
	switch decision {
	case model.ProposalDecisionKeep:
		return model.ProposalSummaryKeep
	case model.ProposalDecisionDefer:
		return model.ProposalSummaryDefer
	default:
		return model.ProposalSummarySkip
	}
}

func traceItem(index int, candidate model.Candidate) model.ProposalItem {
	id := candidate.CapabilityID
	if id == "" {
		id = fmt.Sprintf("b%d", index+1)
	}
	return model.ProposalItem{ID: id, Kind: string(candidate.Execution.Kind), Name: candidate.Description}
}

func newAttempt(started time.Time, raw string, err error, req *Request) model.ProposalAttempt {
	attempt := model.ProposalAttempt{Stage: model.ProposalStageBinding, Status: model.ProposalAttemptSucceeded, DurationMS: time.Since(started).Milliseconds(), RawResponse: raw}
	if req != nil {
		attempt.CapabilityID = req.Capability.CapabilityID
	}
	if err != nil {
		attempt.Status = model.ProposalAttemptFailed
		attempt.Error = &model.RecordError{Code: "proposal_stage_failed", Message: err.Error()}
	}
	return attempt
}
