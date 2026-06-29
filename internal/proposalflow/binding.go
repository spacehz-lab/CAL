package proposalflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func (proposer *LLMProposer) draftBinding(ctx context.Context, req Request, prof profile, capability capabilityPlan, surfaces []surface) (bindingOutput, []byte, caltrace.ProposalStage, error) {
	content, err := proposer.client.Complete(ctx, cliBindingPrompt(req, prof, capability, surfaces))
	if err != nil {
		return bindingOutput{}, nil, caltrace.ProposalStage{}, fmt.Errorf("binding stage: %w", err)
	}
	var output bindingOutput
	if err := json.Unmarshal(content, &output); err != nil {
		return bindingOutput{}, content, caltrace.ProposalStage{}, fmt.Errorf("decode binding stage: %w", err)
	}
	output, stage := normalizeBindings(req, prof, capability, output)
	if len(output.Candidates) == 0 {
		return bindingOutput{}, content, stage, fmt.Errorf("binding stage returned no usable candidates for %q", capability.CapabilityID)
	}
	return output, content, stage, nil
}

func normalizeBindings(req Request, prof profile, capability capabilityPlan, output bindingOutput) (bindingOutput, caltrace.ProposalStage) {
	materials, duplicateMaterials := probeMaterialsByCandidate(output.ProbeMaterials)
	stage := caltrace.ProposalStage{
		Name: caltrace.ProposalStageBinding,
		Summary: map[caltrace.ProposalSummaryKey]int{
			caltrace.ProposalSummaryRaw: len(output.Candidates),
		},
	}
	selected := bindingOutput{}
	for index, candidate := range output.Candidates {
		candidate = normalizeBindingCandidate(candidate)
		traceItem := bindingTraceItem(index, candidate)
		material, ok := materials[index]
		material = normalizeProbeMaterial(material)
		switch {
		case !ok:
			traceItem.Decision = caltrace.ProposalDecisionSkip
			traceItem.Reason = bindingReasonMissingProbeMaterial
		case duplicateMaterials[index]:
			traceItem.Decision = caltrace.ProposalDecisionSkip
			traceItem.Reason = bindingReasonDuplicateProbeMaterial
		default:
			reason := bindingCandidateSkipReason(req, capability, candidate, material)
			switch {
			case reason != "":
				traceItem.Decision = caltrace.ProposalDecisionSkip
				traceItem.Reason = reason
			case prof.maxCandidatesPerCapability > 0 && len(selected.Candidates) >= prof.maxCandidatesPerCapability:
				traceItem.Decision = caltrace.ProposalDecisionDefer
				traceItem.Reason = bindingReasonCandidateLimit
			default:
				traceItem.Decision = caltrace.ProposalDecisionKeep
				selected.Candidates = append(selected.Candidates, candidate)
				material.CandidateIndex = len(selected.ProbeMaterials)
				selected.ProbeMaterials = append(selected.ProbeMaterials, normalizeProbeMaterial(material))
			}
		}
		stage.Items = append(stage.Items, traceItem)
		stage.Summary[summaryKeyForDecision(traceItem.Decision)]++
	}
	stage.Summary[caltrace.ProposalSummarySelected] = len(selected.Candidates)
	return selected, stage
}

func normalizeBindingCandidate(candidate caltrace.Candidate) caltrace.Candidate {
	candidate.ProviderID = strings.TrimSpace(candidate.ProviderID)
	candidate.CapabilityID = strings.TrimSpace(candidate.CapabilityID)
	candidate.Description = strings.TrimSpace(candidate.Description)
	candidate.Source = strings.TrimSpace(candidate.Source)
	return candidate
}

func normalizeProbeMaterial(material probeMaterial) probeMaterial {
	material.Inputs = normalizeProbeInputs(material.Inputs)
	material.Fixtures = normalizeFixtures(material.Fixtures)
	return material
}

func normalizeProbeInputs(inputs map[string]any) map[string]any {
	if len(inputs) == 0 {
		return nil
	}
	normalized := make(map[string]any, len(inputs))
	for key, value := range inputs {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		normalized[key] = value
	}
	return normalized
}

func normalizeFixtures(fixtures []Fixture) []Fixture {
	normalized := make([]Fixture, 0, len(fixtures))
	for _, fixture := range fixtures {
		fixture.Input = strings.TrimSpace(fixture.Input)
		fixture.Filename = strings.TrimSpace(fixture.Filename)
		if fixture.Input == "" && fixture.Filename == "" {
			continue
		}
		normalized = append(normalized, fixture)
	}
	return normalized
}

func probeMaterialsByCandidate(materials []probeMaterial) (map[int]probeMaterial, map[int]bool) {
	byIndex := make(map[int]probeMaterial, len(materials))
	duplicates := map[int]bool{}
	for _, material := range materials {
		if _, ok := byIndex[material.CandidateIndex]; ok {
			duplicates[material.CandidateIndex] = true
			continue
		}
		byIndex[material.CandidateIndex] = material
	}
	return byIndex, duplicates
}

func bindingTraceItem(index int, candidate caltrace.Candidate) caltrace.ProposalItem {
	id := candidate.CapabilityID
	if id == "" {
		id = fmt.Sprintf("b%d", index+1)
	}
	return caltrace.ProposalItem{
		ID:   id,
		Kind: "binding",
		Name: bindingTraceName(candidate),
	}
}

func bindingTraceName(candidate caltrace.Candidate) string {
	args, ok := cliExecutionArgs(candidate.Execution)
	if ok && len(args) > 0 {
		return strings.Join(args, " ")
	}
	if candidate.CapabilityID != "" {
		return candidate.CapabilityID
	}
	return candidate.Description
}

func probeMaterialFor(materials []probeMaterial, index int) (probeMaterial, bool) {
	for _, material := range materials {
		if material.CandidateIndex == index {
			return material, true
		}
	}
	return probeMaterial{}, false
}
