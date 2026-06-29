package proposalflow

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spacehz-lab/cal/internal/core"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func (proposer *LLMProposer) draftEvidence(ctx context.Context, req Request, candidateIndex int, candidate caltrace.Candidate, material probeMaterial) (evidenceOutput, []byte, error) {
	content, err := proposer.client.Complete(ctx, cliEvidencePrompt(req, candidateIndex, candidate, material))
	if err != nil {
		return evidenceOutput{}, nil, fmt.Errorf("evidence stage: %w", err)
	}
	var output evidenceOutput
	if err := json.Unmarshal(content, &output); err != nil {
		return evidenceOutput{}, content, fmt.Errorf("decode evidence stage: %w", err)
	}
	if err := core.ValidateVerifySpec(output.Verify); err != nil {
		return evidenceOutput{}, content, fmt.Errorf("evidence verify spec: %w", err)
	}
	return output, content, nil
}
