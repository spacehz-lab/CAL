package proposal

import (
	"context"
	"fmt"
	"time"

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
	started := time.Now()
	log := newLogger(req.Provider.ID)
	log.proposalStarted(proposer.model)
	if err := ValidatePolicy(proposer.policy); err != nil {
		log.proposalFailed("policy", started, err)
		return Result{}, fmt.Errorf("proposal policy: %w", err)
	}
	prof := selectProfile(req)
	surfaceStarted := time.Now()
	log.stageStarted(caltrace.ProposalStageSurface)
	surfaces, surfaceRaw, surfaceStage, err := proposer.draftSurface(ctx, req, prof)
	if err != nil {
		log.stageFailed(caltrace.ProposalStageSurface, surfaceStarted, err)
		return Result{Diagnostics: proposer.diagnostics(surfaceStage)}, err
	}
	log.stageCompleted(caltrace.ProposalStageSurface, surfaceStarted, logKeySurfaceCount, len(surfaces))

	capabilityStarted := time.Now()
	log.stageStarted(caltrace.ProposalStageCapability, logKeySurfaceCount, len(surfaces))
	capabilities, capabilityRaw, capabilityStage, err := proposer.draftCapabilities(ctx, req, prof, surfaces)
	if err != nil {
		log.stageFailed(caltrace.ProposalStageCapability, capabilityStarted, err)
		return Result{Diagnostics: proposer.diagnostics(surfaceStage, capabilityStage)}, err
	}
	log.stageCompleted(caltrace.ProposalStageCapability, capabilityStarted, logKeyCapabilityCount, len(capabilities))
	raw := append([]byte{}, surfaceRaw...)
	raw = append(raw, capabilityRaw...)
	bindingRun := proposer.newBindingRun(req, prof, surfaces, raw, log)
	result, bindingStages, err := bindingRun.run(ctx, capabilities)
	if err != nil {
		log.proposalFailed(string(caltrace.ProposalStageBinding), started, err)
		return Result{Diagnostics: proposer.diagnostics(append([]caltrace.ProposalStage{surfaceStage, capabilityStage}, bindingStages...)...)}, err
	}
	result.Diagnostics = proposer.diagnostics(append([]caltrace.ProposalStage{surfaceStage, capabilityStage}, bindingStages...)...)
	selected, err := Select(result, SelectOptions{
		ProviderID:  req.Provider.ID,
		DebugFilter: req.DebugFilter,
	})
	if err != nil {
		log.proposalFailed("select", started, err)
		return Result{}, err
	}
	log.proposalCompleted(started, len(selected.Candidates), len(selected.ProbePlans))
	return selected, nil
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
