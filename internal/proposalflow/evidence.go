package proposalflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/runtime"
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
	if !core.ValidVerifierID(output.Verifier.ID) {
		return evidenceOutput{}, content, fmt.Errorf("evidence stage returned invalid verifier id %q", output.Verifier.ID)
	}
	packageIDs := map[string]struct{}{}
	for index, pkg := range output.VerifierPackages {
		if !core.ValidVerifierID(pkg.ID) {
			return evidenceOutput{}, content, fmt.Errorf("evidence verifier package %d id %q is invalid", index, pkg.ID)
		}
		if strings.HasPrefix(pkg.ID, "verifier_") {
			return evidenceOutput{}, content, fmt.Errorf("evidence verifier package id %q must not start with verifier_", pkg.ID)
		}
		if strings.TrimSpace(pkg.VerifyPY) == "" {
			return evidenceOutput{}, content, fmt.Errorf("evidence verifier package %q verify_py is required", pkg.ID)
		}
		packageIDs[pkg.ID] = struct{}{}
	}
	if len(output.VerifierPackages) > 0 {
		if _, ok := packageIDs[output.Verifier.ID]; !ok {
			return evidenceOutput{}, content, fmt.Errorf("evidence verifier %q has no matching verifier package", output.Verifier.ID)
		}
	}
	return output, content, nil
}

func finalVerifier(output evidenceOutput, hash string, capabilityIndex int, candidateIndex int) (core.Verifier, error) {
	verifier := output.Verifier
	if verifier.ID == "" {
		return core.Verifier{}, fmt.Errorf("evidence verifier.id is required")
	}
	for _, pkg := range output.VerifierPackages {
		if pkg.ID != verifier.ID {
			continue
		}
		pkg.ID = generatedVerifierID(hash, capabilityIndex, candidateIndex, pkg.ID)
		if !runtime.DefaultRegistry().Supports(pkg.ID) {
			if err := runtime.InstallVerifier(pkg); err != nil {
				return core.Verifier{}, err
			}
		}
		verifier.ID = pkg.ID
		return verifier, nil
	}
	return verifier, nil
}

func generatedVerifierID(hash string, capabilityIndex int, candidateIndex int, localID string) string {
	return "verifier_" + localID + "_" + core.ShortHash(hash, fmt.Sprint(capabilityIndex), fmt.Sprint(candidateIndex), localID)
}
