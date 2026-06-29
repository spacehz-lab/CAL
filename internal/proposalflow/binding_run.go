package proposalflow

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spacehz-lab/cal/internal/core"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

type bindingRun struct {
	proposer *LLMProposer
	req      Request
	prof     profile
	surfaces []surface
	baseRaw  []byte
	log      logger
}

type bindingRunResult struct {
	result Result
	stages []caltrace.ProposalStage
	err    error
}

func (proposer *LLMProposer) newBindingRun(req Request, prof profile, surfaces []surface, baseRaw []byte, log logger) bindingRun {
	return bindingRun{
		proposer: proposer,
		req:      req,
		prof:     prof,
		surfaces: surfaces,
		baseRaw:  baseRaw,
		log:      log,
	}
}

func (run bindingRun) run(ctx context.Context, capabilities []capabilityPlan) (Result, []caltrace.ProposalStage, error) {
	limit := run.prof.concurrency
	if limit <= 0 {
		limit = 1
	}
	sem := make(chan struct{}, limit)
	results := make([]bindingRunResult, len(capabilities))
	var wg sync.WaitGroup
	var completed atomic.Int64
	started := time.Now()
	run.log.stageStarted(caltrace.ProposalStageBinding,
		logKeyCapabilityCount, len(capabilities),
		logKeyConcurrency, limit,
	)
	for index, capability := range capabilities {
		if err := ctx.Err(); err != nil {
			run.log.stageFailed(caltrace.ProposalStageBinding, started, err,
				logKeyCompleted, completed.Load(),
				logKeyTotal, len(capabilities),
			)
			return Result{}, nil, err
		}
		sem <- struct{}{}
		wg.Add(1)
		go func(index int, capability capabilityPlan) {
			defer wg.Done()
			defer func() { <-sem }()
			pipelineCtx, cancel := run.context(ctx)
			defer cancel()
			pipelineStarted := time.Now()
			run.log.bindingStarted(capability.CapabilityID, index, len(capabilities), run.prof.bindingTimeout)
			results[index] = run.runOne(pipelineCtx, capability, index)
			done := completed.Add(1)
			if results[index].err != nil {
				run.log.bindingFailed(capability.CapabilityID, done, len(capabilities), len(results[index].result.Candidates), pipelineStarted, results[index].err)
				return
			}
			run.log.bindingCompleted(capability.CapabilityID, done, len(capabilities), len(results[index].result.Candidates), pipelineStarted)
		}(index, capability)
	}
	wg.Wait()
	merged, stages, err := mergeBindingRunResults(results)
	if err != nil {
		run.log.stageFailed(caltrace.ProposalStageBinding, started, err,
			logKeyCompleted, completed.Load(),
			logKeyTotal, len(capabilities),
		)
		return merged, stages, err
	}
	run.log.stageCompleted(caltrace.ProposalStageBinding, started,
		logKeyCompleted, completed.Load(),
		logKeyTotal, len(capabilities),
		logKeyCandidateCount, len(merged.Candidates),
	)
	return merged, stages, nil
}

func (run bindingRun) context(ctx context.Context) (context.Context, context.CancelFunc) {
	if run.prof.bindingTimeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, run.prof.bindingTimeout)
}

func (run bindingRun) runOne(ctx context.Context, capability capabilityPlan, capabilityIndex int) bindingRunResult {
	binding, bindingRaw, bindingStage, err := run.proposer.draftBinding(ctx, run.req, run.prof, capability, run.surfaces)
	if err != nil {
		return bindingRunResult{stages: []caltrace.ProposalStage{bindingStage}, err: err}
	}
	raw := append(append([]byte{}, run.baseRaw...), bindingRaw...)
	result := Result{}
	for localIndex, candidate := range binding.Candidates {
		material, ok := probeMaterialFor(binding.ProbeMaterials, localIndex)
		if !ok {
			return bindingRunResult{stages: []caltrace.ProposalStage{bindingStage}, err: fmt.Errorf("binding candidate %d has no probe material", localIndex)}
		}
		candidate = normalizeRunCandidate(run.req.Provider, capability, candidate)
		evidenceStarted := time.Now()
		run.log.evidenceStarted(capability.CapabilityID, localIndex)
		evidence, evidenceRaw, err := run.proposer.draftEvidence(ctx, run.req, localIndex, candidate, material)
		if err != nil {
			run.log.evidenceFailed(capability.CapabilityID, localIndex, evidenceStarted, err)
			return bindingRunResult{stages: []caltrace.ProposalStage{bindingStage}, err: err}
		}
		candidateRaw := append(append([]byte{}, raw...), evidenceRaw...)
		hash := proposalHash(candidateRaw)
		verifier, err := finalVerifier(evidence, hash, capabilityIndex, localIndex)
		if err != nil {
			run.log.evidenceFailed(capability.CapabilityID, localIndex, evidenceStarted, err)
			return bindingRunResult{stages: []caltrace.ProposalStage{bindingStage}, err: err}
		}
		run.log.evidenceCompleted(capability.CapabilityID, localIndex, evidenceStarted, verifier.ID)
		candidate = run.proposer.attachProvenance(candidate, hash)
		result.Candidates = append(result.Candidates, candidate)
		result.ProbePlans = append(result.ProbePlans, ProbePlan{
			CandidateIndex: len(result.Candidates) - 1,
			Inputs:         material.Inputs,
			Fixtures:       material.Fixtures,
			Verifier:       verifier,
		})
	}
	return bindingRunResult{result: result, stages: []caltrace.ProposalStage{bindingStage}}
}

func mergeBindingRunResults(results []bindingRunResult) (Result, []caltrace.ProposalStage, error) {
	var failures []string
	merged := Result{}
	stages := make([]caltrace.ProposalStage, 0, len(results))
	for _, item := range results {
		stages = append(stages, item.stages...)
		if item.err != nil {
			failures = append(failures, item.err.Error())
			continue
		}
		for index, candidate := range item.result.Candidates {
			plan := item.result.ProbePlans[index]
			plan.CandidateIndex = len(merged.Candidates)
			merged.Candidates = append(merged.Candidates, candidate)
			merged.ProbePlans = append(merged.ProbePlans, plan)
		}
	}
	if len(merged.Candidates) == 0 && len(failures) > 0 {
		return Result{}, stages, fmt.Errorf("binding/evidence pipelines failed: %s", strings.Join(failures, "; "))
	}
	return merged, stages, nil
}

func normalizeRunCandidate(provider core.Provider, capability capabilityPlan, candidate caltrace.Candidate) caltrace.Candidate {
	if candidate.ProviderID == "" {
		candidate.ProviderID = provider.ID
	}
	candidate.CapabilityID = capability.CapabilityID
	if candidate.Description == "" {
		candidate.Description = capability.Description
	}
	if candidate.Source == "" {
		candidate.Source = "proposal:" + cliProposalSource
	}
	return candidate
}

func (proposer *LLMProposer) attachProvenance(candidate caltrace.Candidate, hash string) caltrace.Candidate {
	candidate.Provenance = &caltrace.CandidateProvenance{
		Source:        candidate.Source,
		PromptVersion: cliPromptVersion,
		Model:         proposer.model,
		SchemaVersion: cliProposalSchema,
		ProposalHash:  hash,
	}
	return candidate
}
