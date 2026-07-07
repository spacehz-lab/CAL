package rules

import (
	"context"
	"fmt"
	"strings"

	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/proposal"
)

const (
	observationTypeCLIOutput = "cli_output"
	observationContentText   = "text"
)

// Runner proposes candidates from deterministic local observation rules.
type Runner struct{}

// NewRunner creates a deterministic rules proposal runner.
func NewRunner() *Runner {
	return &Runner{}
}

// Run matches observations and returns normalized deterministic candidates.
func (runner *Runner) Run(ctx context.Context, req *proposal.Request) (*proposal.Result, error) {
	if err := runner.validate(req); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	result, err := runner.propose(req)
	if err != nil {
		return nil, err
	}
	normalized, err := proposal.NormalizeResult(result, proposal.NormalizeOptions{
		ProviderID: req.Provider.ID,
	})
	if err != nil {
		return nil, err
	}
	if len(normalized.Candidates) == 0 {
		return normalized, fmt.Errorf("rules proposal produced no matching candidates")
	}
	return normalized, nil
}

func (runner *Runner) validate(req *proposal.Request) error {
	if runner == nil {
		return fmt.Errorf("rules runner is required")
	}
	if req == nil {
		return fmt.Errorf("proposal request is required")
	}
	if req.Provider == nil || strings.TrimSpace(req.Provider.ID) == "" {
		return fmt.Errorf("provider is required")
	}
	return nil
}

func (runner *Runner) propose(req *proposal.Request) (*proposal.Result, error) {
	var candidates []model.Candidate
	for _, observation := range req.Observations {
		if observation.Type != observationTypeCLIOutput {
			continue
		}
		text, ok := observation.Content[observationContentText].(string)
		if !ok {
			continue
		}
		candidate, ok, err := candidateFromHelp(req.Provider.ID, text)
		if err != nil {
			return nil, err
		}
		if ok {
			candidates = append(candidates, candidate)
		}
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("rules proposal matched no observations")
	}
	return resultFromCandidates(candidates), nil
}

func resultFromCandidates(candidates []model.Candidate) *proposal.Result {
	plans := make([]proposal.ProbePlan, 0, len(candidates))
	for index, candidate := range candidates {
		plans = append(plans, proposal.ProbePlan{
			CandidateIndex: index,
			Inputs:         probeInputs(candidate.CapabilityID),
			Fixtures:       probeFixtures(candidate.CapabilityID),
			Verify:         verifyForCapability(candidate.CapabilityID),
		})
	}
	return &proposal.Result{
		Candidates:  candidates,
		ProbePlans:  plans,
		Diagnostics: diagnostics(candidates),
	}
}

func diagnostics(candidates []model.Candidate) *model.ProposalTrace {
	stage := model.ProposalStage{
		Name: model.ProposalStageBinding,
		Summary: map[model.ProposalSummaryKey]int{
			model.ProposalSummaryRaw:      len(candidates),
			model.ProposalSummaryKeep:     len(candidates),
			model.ProposalSummarySelected: len(candidates),
		},
		Items: make([]model.ProposalItem, 0, len(candidates)),
	}
	for _, candidate := range candidates {
		stage.Items = append(stage.Items, model.ProposalItem{
			ID:       candidate.CapabilityID,
			Kind:     string(candidate.Execution.Kind),
			Name:     candidate.Description,
			Decision: model.ProposalDecisionKeep,
			Reason:   candidate.Source,
		})
	}
	return &model.ProposalTrace{
		SchemaVersion: proposal.ProposalSchemaVersion,
		PromptVersion: "rules",
		Model:         "rules",
		Stages:        []model.ProposalStage{stage},
	}
}
