package proposal

import (
	"context"
	"sync"

	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/proposal/binding"
	"github.com/spacehz-lab/cal/internal/proposal/capability"
	"github.com/spacehz-lab/cal/internal/proposal/evidence"
	"github.com/spacehz-lab/cal/internal/proposal/surface"
)

type capabilityPipelineResult struct {
	Index      int
	Candidates []model.Candidate
	ProbePlans []ProbePlan
	Stages     []model.ProposalStage
	Attempts   []model.ProposalAttempt
	Err        error
}

func (runner *Runner) runCapabilityPipelines(ctx context.Context, req *Request, surfaces []surface.Item, plans []capability.Plan, options Options) []capabilityPipelineResult {
	if len(plans) == 0 {
		return nil
	}
	limit := options.Concurrency
	if limit <= 0 || limit > len(plans) {
		limit = len(plans)
	}
	sem := make(chan struct{}, limit)
	results := make([]capabilityPipelineResult, len(plans))
	var wg sync.WaitGroup
	for index, plan := range plans {
		index, plan := index, plan
		results[index].Index = index
		select {
		case <-ctx.Done():
			results[index].Err = ctx.Err()
			continue
		case sem <- struct{}{}:
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			results[index] = runner.runCapabilityPipeline(ctx, req, surfaces, plan, index, options)
		}()
	}
	wg.Wait()
	return results
}

func (runner *Runner) runCapabilityPipeline(ctx context.Context, req *Request, surfaces []surface.Item, plan capability.Plan, capabilityIndex int, options Options) capabilityPipelineResult {
	ctx, cancel := context.WithTimeout(ctx, options.PerCapabilityTimeout)
	defer cancel()

	result := capabilityPipelineResult{Index: capabilityIndex}
	runner.emitStepStarted(ctx, req, model.ProgressStepProposalBinding, plan.CapabilityID, nil)
	bindingResult, err := runner.binding.Run(ctx, &binding.Request{
		Provider:      req.Provider,
		Observations:  req.Observations,
		Surfaces:      bindingSurfacesForPlan(surfaces, plan.SourceSurfaceIDs),
		Capability:    bindingPlan(plan),
		MaxCandidates: options.CandidateLimit,
	})
	bindingStage, bindingAttempt := bindingStepDiagnostics(bindingResult)
	runner.emitStepCompleted(ctx, req, model.ProgressStepProposalBinding, plan.CapabilityID, nil, bindingStage, bindingAttempt, err)
	if bindingResult != nil {
		result.Stages = append(result.Stages, bindingResult.Stage)
		result.Attempts = append(result.Attempts, bindingResult.Attempt)
	}
	if err != nil {
		result.Err = err
		return result
	}

	for index := range bindingResult.Candidates {
		candidate := bindingResult.Candidates[index]
		material := bindingResult.Materials[index]
		runner.emitStepStarted(ctx, req, model.ProgressStepProposalEvidence, candidate.CapabilityID, intPtr(index))
		evidenceResult, err := runner.evidence.Run(ctx, &evidence.Request{
			Provider:       req.Provider,
			Observations:   req.Observations,
			Candidate:      &candidate,
			CandidateIndex: index,
			Material:       evidenceMaterial(material),
		})
		evidenceStage, evidenceAttempt := evidenceStepDiagnostics(evidenceResult)
		runner.emitStepCompleted(ctx, req, model.ProgressStepProposalEvidence, candidate.CapabilityID, intPtr(index), evidenceStage, evidenceAttempt, err)
		if evidenceResult != nil {
			result.Stages = append(result.Stages, evidenceResult.Stage)
			result.Attempts = append(result.Attempts, evidenceResult.Attempt)
		}
		if err != nil {
			result.Err = err
			return result
		}
		result.Candidates = append(result.Candidates, enrichCandidate(candidate, runner.modelName))
		result.ProbePlans = append(result.ProbePlans, probePlan(len(result.Candidates)-1, material, evidenceResult.Verify))
	}
	return result
}

func bindingStepDiagnostics(result *binding.Result) (model.ProposalStage, model.ProposalAttempt) {
	if result == nil {
		return model.ProposalStage{}, model.ProposalAttempt{}
	}
	return result.Stage, result.Attempt
}

func evidenceStepDiagnostics(result *evidence.Result) (model.ProposalStage, model.ProposalAttempt) {
	if result == nil {
		return model.ProposalStage{}, model.ProposalAttempt{}
	}
	return result.Stage, result.Attempt
}

func enrichCandidate(candidate model.Candidate, modelName string) model.Candidate {
	if candidate.Source == "" {
		candidate.Source = candidateSource
	}
	candidate.Provenance = &model.CandidateProvenance{
		Source:        candidate.Source,
		PromptVersion: ProposalPromptVersion,
		Model:         modelName,
		SchemaVersion: ProposalSchemaVersion,
	}
	return candidate
}

func probePlan(candidateIndex int, material binding.ProbeMaterial, verify model.VerifySpec) ProbePlan {
	fixtures := make([]Fixture, 0, len(material.Fixtures))
	for _, fixture := range material.Fixtures {
		fixtures = append(fixtures, proposalFixture(fixture))
	}
	return ProbePlan{CandidateIndex: candidateIndex, Inputs: material.Inputs, Fixtures: fixtures, Verify: verify}
}
