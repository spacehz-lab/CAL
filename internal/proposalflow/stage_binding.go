package proposalflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spacehz-lab/cal/internal/core"
)

func (proposer *LLMProposer) draftBinding(ctx context.Context, req Request, capability capabilityPlanItem, surfaces []surfaceItem) (bindingStageOutput, []byte, error) {
	content, err := proposer.client.Complete(ctx, cliBindingPrompt(req, capability, surfaces))
	if err != nil {
		return bindingStageOutput{}, nil, fmt.Errorf("binding stage: %w", err)
	}
	var output bindingStageOutput
	if err := json.Unmarshal(content, &output); err != nil {
		return bindingStageOutput{}, content, fmt.Errorf("decode binding stage: %w", err)
	}
	if len(output.Candidates) == 0 {
		return bindingStageOutput{}, content, fmt.Errorf("binding stage returned no candidates for %q", capability.CapabilityID)
	}
	for index, candidate := range output.Candidates {
		if candidate.CapabilityID != "" && candidate.CapabilityID != capability.CapabilityID {
			return bindingStageOutput{}, content, fmt.Errorf("binding candidate %d changed capability_id from %q to %q", index, capability.CapabilityID, candidate.CapabilityID)
		}
		if strings.TrimSpace(candidate.Description) == "" && strings.TrimSpace(capability.Description) == "" {
			return bindingStageOutput{}, content, fmt.Errorf("binding candidate %d description is required", index)
		}
		if candidate.Execution.Kind == "" {
			return bindingStageOutput{}, content, fmt.Errorf("binding candidate %d execution is required", index)
		}
		if candidate.Execution.Kind == core.ExecutionKindCLI {
			if _, ok := candidate.Execution.Spec[core.ExecutionSpecArgs]; !ok {
				return bindingStageOutput{}, content, fmt.Errorf("binding candidate %d cli execution args are required", index)
			}
		}
	}
	return output, content, nil
}

func probeMaterialFor(materials []probeMaterial, index int) (probeMaterial, bool) {
	for _, material := range materials {
		if material.CandidateIndex == index {
			return material, true
		}
	}
	return probeMaterial{}, false
}
