package proposal

import (
	"fmt"

	"github.com/spacehz-lab/cal/internal/model"
)

type selectOptions struct {
	ProviderID string
	Limit      int
}

// NormalizeOptions configures mechanical proposal result cleanup.
type NormalizeOptions struct {
	ProviderID string
	Limit      int
}

// NormalizeResult filters, dedupes, limits, and reindexes proposal results.
func NormalizeResult(result *Result, options NormalizeOptions) (*Result, error) {
	return selectResult(result, selectOptions{
		ProviderID: options.ProviderID,
		Limit:      options.Limit,
	})
}

func selectResult(result *Result, options selectOptions) (*Result, error) {
	if result == nil {
		return nil, fmt.Errorf("proposal result is required")
	}
	plansByCandidate := make(map[int]ProbePlan, len(result.ProbePlans))
	for _, plan := range result.ProbePlans {
		if plan.CandidateIndex < 0 || plan.CandidateIndex >= len(result.Candidates) {
			return nil, fmt.Errorf("probe plan candidate_index %d is out of range", plan.CandidateIndex)
		}
		if _, ok := plansByCandidate[plan.CandidateIndex]; ok {
			return nil, fmt.Errorf("candidate %d has duplicate probe plans", plan.CandidateIndex)
		}
		plansByCandidate[plan.CandidateIndex] = plan
	}

	seenExecutions := map[string]struct{}{}
	seenCapabilities := map[string]struct{}{}
	selected := &Result{
		Candidates:  make([]model.Candidate, 0, len(result.Candidates)),
		ProbePlans:  make([]ProbePlan, 0, len(result.Candidates)),
		Diagnostics: result.Diagnostics,
	}
	for index, candidate := range result.Candidates {
		if options.ProviderID != "" && candidate.ProviderID != options.ProviderID {
			continue
		}
		plan, ok := plansByCandidate[index]
		if !ok {
			return nil, fmt.Errorf("candidate %d has no probe plan", index)
		}
		key, err := candidateIdentity(candidate)
		if err != nil {
			return nil, err
		}
		if _, ok := seenExecutions[key]; ok {
			continue
		}
		capabilityKey := candidateCapabilityKey(candidate)
		if _, ok := seenCapabilities[capabilityKey]; ok {
			continue
		}
		if options.Limit > 0 && len(selected.Candidates) >= options.Limit {
			break
		}
		seenExecutions[key] = struct{}{}
		seenCapabilities[capabilityKey] = struct{}{}
		plan.CandidateIndex = len(selected.Candidates)
		selected.Candidates = append(selected.Candidates, candidate)
		selected.ProbePlans = append(selected.ProbePlans, plan)
	}
	return selected, nil
}

func candidateIdentity(candidate model.Candidate) (string, error) {
	canonical, err := model.CanonicalExecution(candidate.Execution)
	if err != nil {
		return "", err
	}
	return candidate.ProviderID + "|" + candidate.CapabilityID + "|" + canonical, nil
}

func candidateCapabilityKey(candidate model.Candidate) string {
	return candidate.ProviderID + "|" + candidate.CapabilityID
}
