package proposal

import (
	"context"
	"fmt"

	"github.com/spacehz-lab/cal/internal/llm"
	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/proposal/binding"
	"github.com/spacehz-lab/cal/internal/proposal/capability"
	"github.com/spacehz-lab/cal/internal/proposal/evidence"
	"github.com/spacehz-lab/cal/internal/proposal/policy"
	"github.com/spacehz-lab/cal/internal/proposal/surface"
)

const candidateSource = "proposal"

// Runner coordinates the four proposal stages.
type Runner struct {
	surface    SurfaceRunner
	capability CapabilityRunner
	binding    BindingRunner
	evidence   EvidenceRunner
	policy     policy.Policy
	options    Options
	modelName  string
}

// NewLiveRunner creates the standard LLM-backed proposal runner.
func NewLiveRunner(client llm.Client, options Options) *Runner {
	return NewWithStages(
		surface.NewRunner(client),
		capability.NewRunner(client),
		binding.NewRunner(client),
		evidence.NewRunner(client),
		options,
	)
}

// NewWithStages creates a proposal runner from explicit stage runners.
func NewWithStages(surface SurfaceRunner, capability CapabilityRunner, binding BindingRunner, evidence EvidenceRunner, options Options) *Runner {
	return &Runner{
		surface:    surface,
		capability: capability,
		binding:    binding,
		evidence:   evidence,
		policy:     policy.Default(),
		options:    normalizeOptions(options),
		modelName:  firstModelName(surface, capability, binding, evidence),
	}
}

// WithPolicy replaces the default proposal policy.
func (runner *Runner) WithPolicy(next policy.Policy) *Runner {
	if runner == nil {
		return runner
	}
	runner.policy = next
	return runner
}

// Run builds candidate bindings and probe plans from observations.
func (runner *Runner) Run(ctx context.Context, req *Request) (*Result, error) {
	if err := runner.validate(req); err != nil {
		return nil, err
	}
	options := normalizeOptions(runner.options)
	ctx, cancel := context.WithTimeout(ctx, options.Timeout)
	defer cancel()

	var stages []model.ProposalStage
	var attempts []model.ProposalAttempt

	runner.emitStepStarted(ctx, req, model.ProgressStepProposalSurface, "", nil)
	surfaceResult, err := runner.surface.Run(ctx, &surface.Request{
		Provider:     req.Provider,
		Observations: req.Observations,
		Policy:       runner.policy.Surface,
		MaxItems:     options.SurfaceLimit,
		Hint:         req.Hint,
	})
	surfaceStage, surfaceAttempt := surfaceStepDiagnostics(surfaceResult)
	runner.emitStepCompleted(ctx, req, model.ProgressStepProposalSurface, "", nil, surfaceStage, surfaceAttempt, err)
	if surfaceResult != nil {
		stages = append(stages, surfaceResult.Stage)
		attempts = append(attempts, surfaceResult.Attempt)
	}
	if err != nil {
		return &Result{Diagnostics: diagnostics(runner.modelName, stages, attempts)}, wrapError(CodeProposalStageFailed, "surface stage failed", err)
	}

	runner.emitStepStarted(ctx, req, model.ProgressStepProposalCapability, "", nil)
	capabilityResult, err := runner.capability.Run(ctx, &capability.Request{
		Provider: req.Provider,
		Surfaces: capabilitySurfaces(surfaceResult.Items),
		Catalog:  req.Catalog,
		Policy:   runner.policy.Capability,
		Hint:     req.Hint,
		MaxPlans: options.CandidateLimit,
	})
	capabilityStage, capabilityAttempt := capabilityStepDiagnostics(capabilityResult)
	runner.emitStepCompleted(ctx, req, model.ProgressStepProposalCapability, "", nil, capabilityStage, capabilityAttempt, err)
	if capabilityResult != nil {
		stages = append(stages, capabilityResult.Stage)
		attempts = append(attempts, capabilityResult.Attempt)
	}
	if err != nil {
		return &Result{Diagnostics: diagnostics(runner.modelName, stages, attempts)}, wrapError(CodeProposalStageFailed, "capability stage failed", err)
	}

	pipelines := runner.runCapabilityPipelines(ctx, req, surfaceResult.Items, capabilityResult.Plans, options)
	result := &Result{Diagnostics: diagnostics(runner.modelName, stages, attempts)}
	var failed int
	for _, pipeline := range pipelines {
		stages = append(stages, pipeline.Stages...)
		attempts = append(attempts, pipeline.Attempts...)
		if pipeline.Err != nil {
			failed++
			continue
		}
		appendPipeline(result, pipeline)
	}
	result.Diagnostics = diagnostics(runner.modelName, stages, attempts)
	if len(result.Candidates) == 0 {
		return result, wrapError(CodeProposalFailed, "proposal produced no candidates", firstPipelineError(pipelines))
	}

	selected, err := selectResult(result, selectOptions{ProviderID: req.Provider.ID, Limit: options.CandidateLimit})
	if err != nil {
		return result, wrapError(CodeProposalSelectFailed, "select proposal candidates", err)
	}
	_ = failed
	return selected, nil
}

func (runner *Runner) validate(req *Request) error {
	if runner == nil {
		return newError(CodeMissingStageRunner, "proposal runner is required")
	}
	if runner.surface == nil || runner.capability == nil || runner.binding == nil || runner.evidence == nil {
		return newError(CodeMissingStageRunner, "all proposal stage runners are required")
	}
	if req == nil {
		return newError(CodeInvalidProposalInput, "proposal request is required")
	}
	if req.Provider == nil || req.Provider.ID == "" {
		return newError(CodeInvalidProposalInput, "provider is required")
	}
	if err := policy.Validate(runner.policy); err != nil {
		return wrapError(CodeInvalidProposalInput, "proposal policy is invalid", err)
	}
	return nil
}

func firstPipelineError(results []capabilityPipelineResult) error {
	for _, result := range results {
		if result.Err != nil {
			return result.Err
		}
	}
	return fmt.Errorf("no capability pipelines ran")
}

func firstModelName(runners ...any) string {
	for _, runner := range runners {
		reporter, ok := runner.(interface{ Model() string })
		if ok && reporter.Model() != "" {
			return reporter.Model()
		}
	}
	return ""
}

func surfaceStepDiagnostics(result *surface.Result) (model.ProposalStage, model.ProposalAttempt) {
	if result == nil {
		return model.ProposalStage{}, model.ProposalAttempt{}
	}
	return result.Stage, result.Attempt
}

func capabilityStepDiagnostics(result *capability.Result) (model.ProposalStage, model.ProposalAttempt) {
	if result == nil {
		return model.ProposalStage{}, model.ProposalAttempt{}
	}
	return result.Stage, result.Attempt
}

func capabilitySurfaces(items []surface.Item) []capability.SurfaceItem {
	result := make([]capability.SurfaceItem, 0, len(items))
	for _, item := range items {
		result = append(result, capability.SurfaceItem{
			ID:          item.ID,
			Kind:        item.Kind,
			Name:        item.Name,
			Description: item.Description,
		})
	}
	return result
}

func bindingSurfacesForPlan(items []surface.Item, sourceIDs []string) []binding.SurfaceItem {
	if len(sourceIDs) == 0 {
		return nil
	}
	selected := map[string]struct{}{}
	for _, id := range sourceIDs {
		if id != "" {
			selected[id] = struct{}{}
		}
	}
	result := make([]binding.SurfaceItem, 0, len(items))
	for _, item := range items {
		if _, ok := selected[item.ID]; !ok {
			continue
		}
		result = append(result, binding.SurfaceItem{
			ID:          item.ID,
			Kind:        item.Kind,
			Name:        item.Name,
			Description: item.Description,
			Usage:       item.Usage,
		})
	}
	return result
}

func bindingPlan(plan capability.Plan) binding.Plan {
	return binding.Plan{
		CapabilityID:     plan.CapabilityID,
		Description:      plan.Description,
		SourceSurfaceIDs: plan.SourceSurfaceIDs,
		Confidence:       plan.Confidence,
	}
}

func appendPipeline(result *Result, pipeline capabilityPipelineResult) {
	offset := len(result.Candidates)
	result.Candidates = append(result.Candidates, pipeline.Candidates...)
	for _, plan := range pipeline.ProbePlans {
		plan.CandidateIndex += offset
		result.ProbePlans = append(result.ProbePlans, plan)
	}
}

func evidenceMaterial(material binding.ProbeMaterial) evidence.Material {
	fixtures := make([]evidence.Fixture, 0, len(material.Fixtures))
	for _, fixture := range material.Fixtures {
		fixtures = append(fixtures, evidence.Fixture{
			Input:    fixture.Input,
			Filename: fixture.Filename,
			Content:  fixture.Content,
		})
	}
	return evidence.Material{Inputs: material.Inputs, Fixtures: fixtures}
}

func proposalFixture(fixture binding.Fixture) Fixture {
	return Fixture{Input: fixture.Input, Filename: fixture.Filename, Content: fixture.Content}
}
