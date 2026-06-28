package proposalflow

import (
	"fmt"

	"github.com/spacehz-lab/cal/internal/core"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

// SelectOptions bounds a Proposal result for one discovery run.
type SelectOptions struct {
	ProviderID  string
	DebugFilter string
	Limit       int
}

// Select filters, de-duplicates, limits, and re-indexes a Proposal result.
func Select(result Result, opts SelectOptions) (Result, error) {
	plansByCandidate := make(map[int]ProbePlan, len(result.ProbePlans))
	for _, plan := range result.ProbePlans {
		if plan.CandidateIndex < 0 || plan.CandidateIndex >= len(result.Candidates) {
			return Result{}, fmt.Errorf("probe plan candidate_index %d is out of range", plan.CandidateIndex)
		}
		if _, exists := plansByCandidate[plan.CandidateIndex]; exists {
			return Result{}, fmt.Errorf("candidate %d has duplicate probe plans", plan.CandidateIndex)
		}
		plansByCandidate[plan.CandidateIndex] = plan
	}

	seen := make(map[string]struct{}, len(result.Candidates))
	selected := Result{
		Candidates: make([]caltrace.Candidate, 0, len(result.Candidates)),
		ProbePlans: make([]ProbePlan, 0, len(result.Candidates)),
	}
	for index, candidate := range result.Candidates {
		if opts.ProviderID != "" && candidate.ProviderID != opts.ProviderID {
			continue
		}
		if opts.DebugFilter != "" && candidate.CapabilityID != opts.DebugFilter {
			continue
		}
		plan, ok := plansByCandidate[index]
		if !ok {
			return Result{}, fmt.Errorf("candidate %d has no probe plan", index)
		}
		key, err := candidateIdentity(candidate)
		if err != nil {
			return Result{}, err
		}
		if _, ok := seen[key]; ok {
			continue
		}
		if opts.Limit > 0 && len(selected.Candidates) >= opts.Limit {
			break
		}
		seen[key] = struct{}{}
		plan.CandidateIndex = len(selected.Candidates)
		selected.Candidates = append(selected.Candidates, candidate)
		selected.ProbePlans = append(selected.ProbePlans, plan)
	}
	return selected, nil
}

func candidateIdentity(candidate caltrace.Candidate) (string, error) {
	canonical, err := core.CanonicalExecution(candidate.Execution)
	if err != nil {
		return "", err
	}
	return candidate.ProviderID + "|" + candidate.CapabilityID + "|" + canonical, nil
}
