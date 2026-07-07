package acquisition

import (
	"context"
	"errors"
	"testing"

	"github.com/spacehz-lab/cal/internal/entry"
	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/observe"
	"github.com/spacehz-lab/cal/internal/probe"
	"github.com/spacehz-lab/cal/internal/promote"
	"github.com/spacehz-lab/cal/internal/proposal"
	"github.com/spacehz-lab/cal/internal/tracelog"
)

func TestRunnerRunCompletesAcquisition(t *testing.T) {
	deps := newDeps()
	runner := deps.runner()

	result, err := runner.Run(context.Background(), request())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Trace.Status != model.TraceStatusCompleted || len(result.Trace.Promotions) != 1 {
		t.Fatalf("trace = %#v, want completed trace with promotion", result.Trace)
	}
	if got := deps.tracer.calls; got != "start,complete" {
		t.Fatalf("trace calls = %q, want start,complete", got)
	}
	if deps.proposer.req.TraceID != "trace_test" || deps.prober.req.TraceID != "trace_test" {
		t.Fatalf("trace ids = %q/%q, want trace_test", deps.proposer.req.TraceID, deps.prober.req.TraceID)
	}
	if deps.prober.req.WorkRoot != "/tmp/cal-trace" {
		t.Fatalf("probe work root = %q, want request work root", deps.prober.req.WorkRoot)
	}
	if deps.proposer.req.Hint != "convert pdf" {
		t.Fatalf("proposal hint = %q, want convert pdf", deps.proposer.req.Hint)
	}
}

func TestRunnerRunEmitsProgress(t *testing.T) {
	deps := newDeps()
	var events []model.ProgressEvent
	runner := deps.runner(WithProgress(func(_ context.Context, event *model.ProgressEvent) {
		events = append(events, *event)
	}))

	if _, err := runner.Run(context.Background(), request()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	assertProgress(t, events, []progressWant{
		{model.ProgressStageEntry, model.ProgressStatusStarted},
		{model.ProgressStageEntry, model.ProgressStatusSucceeded},
		{model.ProgressStageCatalog, model.ProgressStatusStarted},
		{model.ProgressStageCatalog, model.ProgressStatusSucceeded},
		{model.ProgressStageObserve, model.ProgressStatusStarted},
		{model.ProgressStageObserve, model.ProgressStatusSucceeded},
		{model.ProgressStageProposal, model.ProgressStatusStarted},
		{model.ProgressStageProposal, model.ProgressStatusSucceeded},
		{model.ProgressStageProbe, model.ProgressStatusStarted},
		{model.ProgressStageProbe, model.ProgressStatusSucceeded},
		{model.ProgressStagePromote, model.ProgressStatusStarted},
		{model.ProgressStagePromote, model.ProgressStatusSucceeded},
	})
}

func TestRunnerRunProviderLoadFailureDoesNotWriteTrace(t *testing.T) {
	deps := newDeps()
	deps.loader.err = errors.New("missing provider")
	var events []model.ProgressEvent
	runner := deps.runner(WithProgress(func(_ context.Context, event *model.ProgressEvent) {
		events = append(events, *event)
	}))

	_, err := runner.Run(context.Background(), request())
	if err == nil {
		t.Fatal("Run() error = nil, want provider load error")
	}
	var acquisitionErr *Error
	if !errors.As(err, &acquisitionErr) || acquisitionErr.Code != CodeProviderLoadFailed {
		t.Fatalf("Run() error = %#v, want CodeProviderLoadFailed", err)
	}
	if deps.tracer.calls != "" {
		t.Fatalf("trace calls = %q, want none", deps.tracer.calls)
	}
	assertProgress(t, events, []progressWant{
		{model.ProgressStageEntry, model.ProgressStatusStarted},
		{model.ProgressStageEntry, model.ProgressStatusFailed},
	})
}

func TestRunnerRunProposalFailureWritesPartialTrace(t *testing.T) {
	deps := newDeps()
	deps.proposer.err = errors.New("proposal failed")
	deps.proposer.result = &proposal.Result{
		Diagnostics: &model.ProposalTrace{Model: "gpt-test"},
		Candidates:  []model.Candidate{candidate()},
		ProbePlans:  []proposal.ProbePlan{probePlan()},
	}
	runner := deps.runner()

	result, err := runner.Run(context.Background(), request())
	if err == nil {
		t.Fatal("Run() error = nil, want proposal error")
	}
	var acquisitionErr *Error
	if !errors.As(err, &acquisitionErr) || acquisitionErr.Code != CodeProposalFailed {
		t.Fatalf("Run() error = %#v, want CodeProposalFailed", err)
	}
	if result == nil || result.Trace.Status != model.TraceStatusFailed || result.Trace.Error.Code != CodeProposalFailed {
		t.Fatalf("result = %#v, want failed trace with proposal error", result)
	}
	if len(result.Trace.Candidates) != 1 || result.Trace.Proposal == nil {
		t.Fatalf("trace = %#v, want partial proposal state", result.Trace)
	}
	if got := deps.tracer.calls; got != "start,fail" {
		t.Fatalf("trace calls = %q, want start,fail", got)
	}
}

func TestRunnerRunCanceledContextWritesCanceledTrace(t *testing.T) {
	deps := newDeps()
	deps.observer.cancelContext = true
	var events []model.ProgressEvent
	runner := deps.runner(WithProgress(func(_ context.Context, event *model.ProgressEvent) {
		events = append(events, *event)
	}))

	result, err := runner.Run(context.Background(), request())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %#v, want context.Canceled", err)
	}
	if result == nil || result.Trace.Status != model.TraceStatusCanceled || result.Trace.Error.Code != CodeObserveFailed {
		t.Fatalf("result = %#v, want canceled trace with observe error", result)
	}
	if got := deps.tracer.calls; got != "start,cancel" {
		t.Fatalf("trace calls = %q, want start,cancel", got)
	}
	if len(events) == 0 || events[len(events)-1].Stage != model.ProgressStageObserve || events[len(events)-1].Status != model.ProgressStatusCanceled {
		t.Fatalf("events = %#v, want final observe canceled", events)
	}
}

func TestRunnerRunTerminalTraceWriteFailurePreservesStageError(t *testing.T) {
	deps := newDeps()
	catalogErr := errors.New("catalog failed")
	deps.catalog.err = catalogErr
	deps.tracer.failErr = errors.New("trace failed")
	runner := deps.runner()

	_, err := runner.Run(context.Background(), request())
	if err == nil {
		t.Fatal("Run() error = nil, want trace write error")
	}
	var acquisitionErr *Error
	if !errors.As(err, &acquisitionErr) || acquisitionErr.Code != CodeTraceWriteFailed {
		t.Fatalf("Run() error = %#v, want CodeTraceWriteFailed", err)
	}
	if !errors.Is(err, catalogErr) {
		t.Fatalf("Run() error = %#v, want wrapped catalog error", err)
	}
}

func TestRunnerRunRejectsInvalidInput(t *testing.T) {
	deps := newDeps()
	runner := deps.runner()
	if _, err := runner.Run(context.Background(), nil); err == nil {
		t.Fatal("Run() error = nil, want nil request error")
	}
	if _, err := NewRunner(nil, deps.catalog, deps.observer, deps.proposer, deps.prober, deps.promoter, deps.tracer).Run(context.Background(), request()); err == nil {
		t.Fatal("Run() error = nil, want missing dependency error")
	}
}

func request() *Request {
	return &Request{
		ProviderID: "provider_cli",
		Hint:       "convert pdf",
		WorkRoot:   "/tmp/cal-trace",
	}
}

type deps struct {
	loader   *fakeLoader
	catalog  *fakeCatalog
	observer *fakeObserver
	proposer *fakeProposer
	prober   *fakeProber
	promoter *fakePromoter
	tracer   *fakeTracer
}

func newDeps() *deps {
	return &deps{
		loader:   &fakeLoader{result: &entry.LoadResult{Provider: provider()}},
		catalog:  &fakeCatalog{capabilities: []model.Capability{{ID: "document.convert", Description: "Convert a document."}}},
		observer: &fakeObserver{result: &observe.Result{ProviderID: "provider_cli", Observations: []model.Observation{observation()}}},
		proposer: &fakeProposer{result: &proposal.Result{Diagnostics: &model.ProposalTrace{Model: "gpt-test"}, Candidates: []model.Candidate{candidate()}, ProbePlans: []proposal.ProbePlan{probePlan()}}},
		prober:   &fakeProber{result: &probe.Result{Probes: []model.Probe{passedProbe()}}},
		promoter: &fakePromoter{result: &promote.Result{Promotions: []model.Promotion{promotion()}}},
		tracer:   &fakeTracer{},
	}
}

func (deps *deps) runner(opts ...Option) *Runner {
	return NewRunner(deps.loader, deps.catalog, deps.observer, deps.proposer, deps.prober, deps.promoter, deps.tracer, opts...)
}

type progressWant struct {
	stage  model.ProgressStage
	status model.ProgressStatus
}

func assertProgress(t *testing.T, events []model.ProgressEvent, wants []progressWant) {
	t.Helper()
	if len(events) != len(wants) {
		t.Fatalf("progress len = %d, want %d: %#v", len(events), len(wants), events)
	}
	for index, want := range wants {
		event := events[index]
		if event.Scope != model.ProgressScopeAcquisition || event.Stage != want.stage || event.Status != want.status {
			t.Fatalf("event[%d] = %#v, want %s/%s", index, event, want.stage, want.status)
		}
		if event.ID == "" || event.CreatedAt == "" {
			t.Fatalf("event[%d] = %#v, want id and created_at", index, event)
		}
	}
}

type fakeLoader struct {
	result *entry.LoadResult
	err    error
}

func (loader *fakeLoader) Load(_ context.Context, req *entry.LoadRequest) (*entry.LoadResult, error) {
	if req.ProviderID != "provider_cli" {
		return nil, errors.New("unexpected provider id")
	}
	return loader.result, loader.err
}

type fakeCatalog struct {
	capabilities []model.Capability
	err          error
}

func (catalog *fakeCatalog) ListCapabilities() ([]model.Capability, error) {
	return catalog.capabilities, catalog.err
}

type fakeObserver struct {
	result        *observe.Result
	err           error
	cancelContext bool
}

func (observer *fakeObserver) Observe(ctx context.Context, req *observe.Request) (*observe.Result, error) {
	if req.Provider == nil || req.Provider.ID != "provider_cli" {
		return nil, errors.New("unexpected observe provider")
	}
	if observer.cancelContext {
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		return nil, canceled.Err()
	}
	return observer.result, observer.err
}

type fakeProposer struct {
	req    *proposal.Request
	result *proposal.Result
	err    error
}

func (proposer *fakeProposer) Run(_ context.Context, req *proposal.Request) (*proposal.Result, error) {
	proposer.req = req
	return proposer.result, proposer.err
}

type fakeProber struct {
	req    *probe.Request
	result *probe.Result
	err    error
}

func (prober *fakeProber) Run(_ context.Context, req *probe.Request) (*probe.Result, error) {
	prober.req = req
	return prober.result, prober.err
}

type fakePromoter struct {
	req    *promote.Request
	result *promote.Result
	err    error
}

func (promoter *fakePromoter) Run(_ context.Context, req *promote.Request) (*promote.Result, error) {
	promoter.req = req
	return promoter.result, promoter.err
}

type fakeTracer struct {
	calls   string
	failErr error
}

func (tracer *fakeTracer) Start(_ context.Context, req *tracelog.Request) (*tracelog.Result, error) {
	tracer.addCall("start")
	trace := traceFrom(req, model.TraceStatusRunning)
	trace.ID = "trace_test"
	trace.StartedAt = "2026-01-01T00:00:00Z"
	return &tracelog.Result{Trace: trace}, nil
}

func (tracer *fakeTracer) Complete(_ context.Context, req *tracelog.Request) (*tracelog.Result, error) {
	tracer.addCall("complete")
	return &tracelog.Result{Trace: traceFrom(req, model.TraceStatusCompleted)}, nil
}

func (tracer *fakeTracer) Fail(_ context.Context, req *tracelog.Request) (*tracelog.Result, error) {
	tracer.addCall("fail")
	if tracer.failErr != nil {
		return nil, tracer.failErr
	}
	return &tracelog.Result{Trace: traceFrom(req, model.TraceStatusFailed)}, nil
}

func (tracer *fakeTracer) Cancel(ctx context.Context, req *tracelog.Request) (*tracelog.Result, error) {
	tracer.addCall("cancel")
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &tracelog.Result{Trace: traceFrom(req, model.TraceStatusCanceled)}, nil
}

func (tracer *fakeTracer) addCall(call string) {
	if tracer.calls == "" {
		tracer.calls = call
		return
	}
	tracer.calls += "," + call
}

func traceFrom(req *tracelog.Request, status model.TraceStatus) model.Trace {
	return model.Trace{
		ID:           req.TraceID,
		StartedAt:    req.StartedAt,
		EndedAt:      "2026-01-01T00:00:01Z",
		Status:       status,
		Hint:         req.Hint,
		ProviderIDs:  req.ProviderIDs,
		Observations: req.Observations,
		Proposal:     req.Proposal,
		Candidates:   req.Candidates,
		Probes:       req.Probes,
		Promotions:   req.Promotions,
		Error:        req.Error,
	}
}

func provider() model.Provider {
	return model.Provider{ID: "provider_cli", Kind: model.ProviderKindCLI, Path: "/bin/test"}
}

func observation() model.Observation {
	return model.Observation{ProviderID: "provider_cli", Type: observe.ObservationTypeCLIOutput}
}

func candidate() model.Candidate {
	return model.Candidate{
		ProviderID:   "provider_cli",
		CapabilityID: "document.convert",
		Description:  "Convert a document.",
		Execution:    model.Execution{Kind: model.ExecutionKindCLI},
	}
}

func probePlan() proposal.ProbePlan {
	return proposal.ProbePlan{
		CandidateIndex: 0,
		Verify: model.VerifySpec{
			Level:  model.VerifyLevelL1,
			Method: model.VerifyMethodExecute,
		},
	}
}

func passedProbe() model.Probe {
	return model.Probe{
		CandidateIndex: 0,
		Passed:         true,
		Verify:         probePlan().Verify,
		Evidence:       []model.EvidenceRef{{ID: "evidence_file_exists"}},
	}
}

func promotion() model.Promotion {
	return model.Promotion{
		CandidateIndex: 0,
		CapabilityID:   "document.convert",
		BindingID:      "binding_abc",
		ProviderID:     "provider_cli",
	}
}
