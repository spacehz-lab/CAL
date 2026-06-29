package proposalflow

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/spacehz-lab/cal/internal/core"
	sharedllm "github.com/spacehz-lab/cal/internal/llm"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

// LLMProposer runs the live four-stage Proposal flow.
type LLMProposer struct {
	client sharedllm.Client
	model  string
	policy Policy
}

type modelReporter interface {
	Model() string
}

// NewLLMProposer builds a live LLM-backed Proposal proposer.
func NewLLMProposer(client sharedllm.Client) *LLMProposer {
	return NewLLMProposerWithPolicy(client, DefaultPolicy())
}

// NewLLMProposerWithPolicy builds a live LLM-backed Proposal proposer with local policy.
func NewLLMProposerWithPolicy(client sharedllm.Client, policy Policy) *LLMProposer {
	return &LLMProposer{
		client: client,
		model:  modelOf(client),
		policy: policy,
	}
}

// Propose runs Surface, Capability, Binding, and Evidence, then returns final executable proposal material.
func (proposer *LLMProposer) Propose(ctx context.Context, req Request) (Result, error) {
	if proposer == nil || proposer.client == nil {
		return Result{}, sharedllm.ErrNoClient
	}
	if err := ValidatePolicy(proposer.policy); err != nil {
		return Result{}, fmt.Errorf("proposal policy: %w", err)
	}
	prof := selectProfile(req)
	surfaces, surfaceRaw, surfaceStage, err := proposer.extractSurface(ctx, req, prof)
	if err != nil {
		return Result{Diagnostics: proposer.diagnostics(surfaceStage)}, err
	}
	capabilities, capabilityRaw, err := proposer.planCapabilities(ctx, req, prof, surfaces)
	if err != nil {
		return Result{Diagnostics: proposer.diagnostics(surfaceStage)}, err
	}
	raw := append([]byte{}, surfaceRaw...)
	raw = append(raw, capabilityRaw...)
	result, err := proposer.proposeBindings(ctx, req, prof, surfaces, capabilities, raw)
	if err != nil {
		return Result{Diagnostics: proposer.diagnostics(surfaceStage)}, err
	}
	result.Diagnostics = proposer.diagnostics(surfaceStage)
	return Select(result, SelectOptions{
		ProviderID:  req.Provider.ID,
		DebugFilter: req.DebugFilter,
	})
}

func (proposer *LLMProposer) diagnostics(stages ...caltrace.ProposalStage) *caltrace.ProposalTrace {
	filtered := make([]caltrace.ProposalStage, 0, len(stages))
	for _, stage := range stages {
		if stage.Name != "" {
			filtered = append(filtered, stage)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return &caltrace.ProposalTrace{
		SchemaVersion: cliProposalSchema,
		PromptVersion: cliPromptVersion,
		Model:         proposer.model,
		Stages:        filtered,
	}
}

func modelOf(client sharedllm.Client) string {
	if reporter, ok := client.(modelReporter); ok {
		return reporter.Model()
	}
	return ""
}

func normalizeProposedCandidate(provider core.Provider, capability capabilityPlanItem, candidate caltrace.Candidate) caltrace.Candidate {
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

type bindingResult struct {
	result Result
	err    error
}

func (proposer *LLMProposer) proposeBindings(ctx context.Context, req Request, prof profile, surfaces []surfaceItem, capabilities []capabilityPlanItem, baseRaw []byte) (Result, error) {
	limit := prof.concurrency
	if limit <= 0 {
		limit = 1
	}
	sem := make(chan struct{}, limit)
	results := make([]bindingResult, len(capabilities))
	var wg sync.WaitGroup
	for index, capability := range capabilities {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		sem <- struct{}{}
		wg.Add(1)
		go func(index int, capability capabilityPlanItem) {
			defer wg.Done()
			defer func() { <-sem }()
			results[index] = proposer.proposeBinding(ctx, req, surfaces, capability, index, baseRaw)
		}(index, capability)
	}
	wg.Wait()
	return mergeBindingResults(results)
}

func (proposer *LLMProposer) proposeBinding(ctx context.Context, req Request, surfaces []surfaceItem, capability capabilityPlanItem, capabilityIndex int, baseRaw []byte) bindingResult {
	binding, bindingRaw, err := proposer.draftBinding(ctx, req, capability, surfaces)
	if err != nil {
		return bindingResult{err: err}
	}
	raw := append(append([]byte{}, baseRaw...), bindingRaw...)
	result := Result{}
	for localIndex, candidate := range binding.Candidates {
		material, ok := probeMaterialFor(binding.ProbeMaterials, localIndex)
		if !ok {
			return bindingResult{err: fmt.Errorf("binding candidate %d has no probe material", localIndex)}
		}
		candidate = normalizeProposedCandidate(req.Provider, capability, candidate)
		evidence, evidenceRaw, err := proposer.planEvidence(ctx, req, localIndex, candidate, material)
		if err != nil {
			return bindingResult{err: err}
		}
		candidateRaw := append(append([]byte{}, raw...), evidenceRaw...)
		hash := proposalHash(candidateRaw)
		verifier, err := finalVerifier(evidence, hash, capabilityIndex, localIndex)
		if err != nil {
			return bindingResult{err: err}
		}
		candidate = proposer.attachProvenance(candidate, hash)
		result.Candidates = append(result.Candidates, candidate)
		result.ProbePlans = append(result.ProbePlans, ProbePlan{
			CandidateIndex: len(result.Candidates) - 1,
			Inputs:         material.Inputs,
			Fixtures:       material.Fixtures,
			Verifier:       verifier,
		})
	}
	return bindingResult{result: result}
}

func mergeBindingResults(results []bindingResult) (Result, error) {
	var failures []string
	merged := Result{}
	for _, item := range results {
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
		return Result{}, fmt.Errorf("binding/evidence pipelines failed: %s", strings.Join(failures, "; "))
	}
	return merged, nil
}
