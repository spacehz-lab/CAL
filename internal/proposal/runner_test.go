package proposal

import (
	"context"
	"errors"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/progress"
	"github.com/spacehz-lab/cal/internal/proposal/binding"
	"github.com/spacehz-lab/cal/internal/proposal/capability"
	"github.com/spacehz-lab/cal/internal/proposal/evidence"
	"github.com/spacehz-lab/cal/internal/proposal/surface"
)

func TestRunnerRunMergesCapabilityPipelinesWithGlobalCandidateIndexes(t *testing.T) {
	runner := NewWithStages(
		fakeSurface{items: []surface.Item{{ID: "s1", Kind: "command", Name: "convert"}}},
		fakeCapability{plans: []capability.Plan{
			{CapabilityID: "pdf.convert", SourceSurfaceIDs: []string{"s1"}},
			{CapabilityID: "image.resize", SourceSurfaceIDs: []string{"s1"}},
		}},
		fakeBinding{},
		fakeEvidence{},
		Options{Concurrency: 2},
	)

	result, err := runner.Run(context.Background(), &Request{Provider: &model.Provider{ID: "provider_test"}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(result.Candidates) != 2 {
		t.Fatalf("len(Candidates) = %d, want 2", len(result.Candidates))
	}
	if got := []int{result.ProbePlans[0].CandidateIndex, result.ProbePlans[1].CandidateIndex}; got[0] != 0 || got[1] != 1 {
		t.Fatalf("ProbePlan indexes = %v, want [0 1]", got)
	}
}

func TestRunnerRunKeepsSuccessfulPipelineWhenAnotherCapabilityFails(t *testing.T) {
	runner := NewWithStages(
		fakeSurface{items: []surface.Item{{ID: "s1", Kind: "command", Name: "convert"}}},
		fakeCapability{plans: []capability.Plan{
			{CapabilityID: "pdf.convert", SourceSurfaceIDs: []string{"s1"}},
			{CapabilityID: "image.resize", SourceSurfaceIDs: []string{"s1"}},
		}},
		fakeBinding{failCapabilityID: "image.resize"},
		fakeEvidence{},
		Options{Concurrency: 2},
	)

	result, err := runner.Run(context.Background(), &Request{Provider: &model.Provider{ID: "provider_test"}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("len(Candidates) = %d, want 1", len(result.Candidates))
	}
	if result.Candidates[0].CapabilityID != "pdf.convert" {
		t.Fatalf("CapabilityID = %q, want pdf.convert", result.Candidates[0].CapabilityID)
	}
}

func TestRunnerRunReturnsProposalErrorWhenEveryPipelineFails(t *testing.T) {
	runner := NewWithStages(
		fakeSurface{items: []surface.Item{{ID: "s1", Kind: "command", Name: "convert"}}},
		fakeCapability{plans: []capability.Plan{{CapabilityID: "pdf.convert", SourceSurfaceIDs: []string{"s1"}}}},
		fakeBinding{failCapabilityID: "pdf.convert"},
		fakeEvidence{},
		Options{},
	)

	result, err := runner.Run(context.Background(), &Request{Provider: &model.Provider{ID: "provider_test"}})
	if err == nil {
		t.Fatal("Run() error = nil, want error")
	}
	var proposalErr *Error
	if !errors.As(err, &proposalErr) || proposalErr.Code != CodeProposalFailed {
		t.Fatalf("Run() error = %#v, want CodeProposalFailed", err)
	}
	if result == nil || result.Diagnostics == nil {
		t.Fatalf("Run() result diagnostics = nil, want partial diagnostics")
	}
}

func TestRunnerRunValidatesStageRunners(t *testing.T) {
	runner := NewWithStages(nil, fakeCapability{}, fakeBinding{}, fakeEvidence{}, Options{})

	_, err := runner.Run(context.Background(), &Request{Provider: &model.Provider{ID: "provider_test"}})
	if err == nil {
		t.Fatal("Run() error = nil, want validation error")
	}
	var proposalErr *Error
	if !errors.As(err, &proposalErr) || proposalErr.Code != CodeMissingStageRunner {
		t.Fatalf("Run() error = %#v, want CodeMissingStageRunner", err)
	}
}

func TestRunnerRunEmitsProposalStepProgress(t *testing.T) {
	var events []model.ProgressEvent
	ctx := progress.WithHandler(context.Background(), func(_ context.Context, event *model.ProgressEvent) {
		events = append(events, *event)
	})
	runner := NewWithStages(
		fakeSurface{items: []surface.Item{{ID: "s1", Kind: "command", Name: "convert"}}},
		fakeCapability{plans: []capability.Plan{{CapabilityID: "pdf.convert", SourceSurfaceIDs: []string{"s1"}}}},
		fakeBinding{},
		fakeEvidence{},
		Options{Concurrency: 1},
	)

	_, err := runner.Run(ctx, &Request{Provider: &model.Provider{ID: "provider_test"}, TraceID: "trace_test"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	want := []struct {
		step   model.ProgressStep
		status model.ProgressStatus
	}{
		{model.ProgressStepProposalSurface, model.ProgressStatusStarted},
		{model.ProgressStepProposalSurface, model.ProgressStatusSucceeded},
		{model.ProgressStepProposalCapability, model.ProgressStatusStarted},
		{model.ProgressStepProposalCapability, model.ProgressStatusSucceeded},
		{model.ProgressStepProposalBinding, model.ProgressStatusStarted},
		{model.ProgressStepProposalBinding, model.ProgressStatusSucceeded},
		{model.ProgressStepProposalEvidence, model.ProgressStatusStarted},
		{model.ProgressStepProposalEvidence, model.ProgressStatusSucceeded},
	}
	if len(events) != len(want) {
		t.Fatalf("events = %#v, want %d", events, len(want))
	}
	for index, item := range want {
		event := events[index]
		if event.Scope != model.ProgressScopeAcquisition || event.Stage != model.ProgressStageProposal || event.Step != item.step || event.Status != item.status || event.TraceID != "trace_test" || event.ProviderID != "provider_test" {
			t.Fatalf("event[%d] = %#v, want %s/%s", index, event, item.step, item.status)
		}
	}
	if events[5].Details["raw_response"] != "binding raw" || events[5].Details["raw_response_bytes"] != len("binding raw") {
		t.Fatalf("binding details = %#v, want raw response diagnostics", events[5].Details)
	}
}

func TestRunnerRunUsesDefaultSurfaceLimit(t *testing.T) {
	surfaceRunner := &captureSurface{items: []surface.Item{{ID: "s1", Kind: "command", Name: "convert"}}}
	runner := NewWithStages(
		surfaceRunner,
		fakeCapability{plans: []capability.Plan{{CapabilityID: "pdf.convert", SourceSurfaceIDs: []string{"s1"}}}},
		fakeBinding{},
		fakeEvidence{},
		Options{},
	)

	_, err := runner.Run(context.Background(), &Request{Provider: &model.Provider{ID: "provider_test"}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if surfaceRunner.request == nil || surfaceRunner.request.MaxItems != DefaultMaxSurfaceItems {
		t.Fatalf("surface MaxItems = %#v, want %d", surfaceRunner.request, DefaultMaxSurfaceItems)
	}
}

func TestRunnerRunPassesOnlySourceSurfacesToBinding(t *testing.T) {
	bindingRunner := &captureBinding{}
	runner := NewWithStages(
		fakeSurface{items: []surface.Item{
			{ID: "s1", Kind: "subcommand", Name: "make-pdf", Usage: "make-pdf --in <input> --out <output>"},
			{ID: "s2", Kind: "subcommand", Name: "write-note", Usage: "write-note --in <input> --out <output>"},
		}},
		fakeCapability{plans: []capability.Plan{{CapabilityID: "file.write", SourceSurfaceIDs: []string{"s2"}}}},
		bindingRunner,
		fakeEvidence{},
		Options{},
	)

	_, err := runner.Run(context.Background(), &Request{Provider: &model.Provider{ID: "provider_test"}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if bindingRunner.request == nil {
		t.Fatal("binding request = nil")
	}
	if len(bindingRunner.request.Surfaces) != 1 {
		t.Fatalf("binding surfaces = %#v, want one selected surface", bindingRunner.request.Surfaces)
	}
	got := bindingRunner.request.Surfaces[0]
	if got.ID != "s2" || got.Usage != "write-note --in <input> --out <output>" {
		t.Fatalf("binding surface = %#v, want source surface s2 with usage", got)
	}
}

type fakeSurface struct {
	items []surface.Item
	err   error
}

func (fake fakeSurface) Run(context.Context, *surface.Request) (*surface.Result, error) {
	return &surface.Result{Items: fake.items, Stage: model.ProposalStage{Name: model.ProposalStageSurface}, Attempt: model.ProposalAttempt{Stage: model.ProposalStageSurface}}, fake.err
}

type captureSurface struct {
	items   []surface.Item
	request *surface.Request
}

func (fake *captureSurface) Run(_ context.Context, req *surface.Request) (*surface.Result, error) {
	fake.request = req
	return &surface.Result{Items: fake.items, Stage: model.ProposalStage{Name: model.ProposalStageSurface}, Attempt: model.ProposalAttempt{Stage: model.ProposalStageSurface}}, nil
}

type fakeCapability struct {
	plans []capability.Plan
	err   error
}

func (fake fakeCapability) Run(context.Context, *capability.Request) (*capability.Result, error) {
	return &capability.Result{Plans: fake.plans, Stage: model.ProposalStage{Name: model.ProposalStageCapability}, Attempt: model.ProposalAttempt{Stage: model.ProposalStageCapability}}, fake.err
}

type fakeBinding struct {
	failCapabilityID string
}

func (fake fakeBinding) Run(_ context.Context, req *binding.Request) (*binding.Result, error) {
	if req.Capability.CapabilityID == fake.failCapabilityID {
		return &binding.Result{Attempt: model.ProposalAttempt{Stage: model.ProposalStageBinding, CapabilityID: req.Capability.CapabilityID}}, errors.New("binding failed")
	}
	candidate := model.Candidate{
		ProviderID:   req.Provider.ID,
		CapabilityID: req.Capability.CapabilityID,
		Description:  req.Capability.Description,
		Execution:    model.Execution{Kind: model.ExecutionKindCLI, Spec: map[string]any{model.ExecutionSpecArgs: []any{"cal-test", req.Capability.CapabilityID}}},
	}
	material := binding.ProbeMaterial{CandidateIndex: 0, Inputs: map[string]any{"sample": "value"}}
	return &binding.Result{
		Candidates: []model.Candidate{candidate},
		Materials:  []binding.ProbeMaterial{material},
		Stage:      model.ProposalStage{Name: model.ProposalStageBinding},
		Attempt:    model.ProposalAttempt{Stage: model.ProposalStageBinding, CapabilityID: req.Capability.CapabilityID, RawResponse: "binding raw"},
	}, nil
}

type captureBinding struct {
	request *binding.Request
}

func (fake *captureBinding) Run(ctx context.Context, req *binding.Request) (*binding.Result, error) {
	fake.request = req
	return fakeBinding{}.Run(ctx, req)
}

type fakeEvidence struct{}

func (fakeEvidence) Run(_ context.Context, req *evidence.Request) (*evidence.Result, error) {
	verify := model.VerifySpec{
		Level:  model.VerifyLevelL1,
		Method: model.VerifyMethodExecute,
		Checks: []model.VerifyCheck{{Subject: model.VerifySubject{Type: model.VerifySubjectStdout}, Predicate: model.VerifyPredicateNonEmpty}},
	}
	return &evidence.Result{
		Verify:  verify,
		Stage:   model.ProposalStage{Name: model.ProposalStageEvidence},
		Attempt: model.ProposalAttempt{Stage: model.ProposalStageEvidence, CandidateIndex: &req.CandidateIndex},
	}, nil
}
