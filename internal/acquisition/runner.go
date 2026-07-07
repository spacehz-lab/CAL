package acquisition

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/spacehz-lab/cal/internal/entry"
	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/observe"
	"github.com/spacehz-lab/cal/internal/probe"
	"github.com/spacehz-lab/cal/internal/progress"
	"github.com/spacehz-lab/cal/internal/promote"
	"github.com/spacehz-lab/cal/internal/proposal"
	"github.com/spacehz-lab/cal/internal/tracelog"
)

// Runner orchestrates one acquisition flow.
type Runner struct {
	loader   ProviderLoader
	catalog  CatalogStore
	observer Observer
	proposer Proposer
	prober   Prober
	promoter Promoter
	tracer   TraceWriter

	onProgress ProgressFunc
}

// NewRunner creates an acquisition runner.
func NewRunner(loader ProviderLoader, catalog CatalogStore, observer Observer, proposer Proposer, prober Prober, promoter Promoter, tracer TraceWriter, opts ...Option) *Runner {
	runner := &Runner{
		loader:   loader,
		catalog:  catalog,
		observer: observer,
		proposer: proposer,
		prober:   prober,
		promoter: promoter,
		tracer:   tracer,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(runner)
		}
	}
	return runner
}

// Run executes one acquisition flow.
func (runner *Runner) Run(ctx context.Context, req *Request) (*Result, error) {
	ctx = normalizeContext(ctx)
	if err := runner.validate(req); err != nil {
		return nil, err
	}

	st := &state{}
	stageStarted := time.Now().UTC()
	runner.emitProgress(ctx, req, st, model.ProgressStageEntry, model.ProgressStatusStarted, stageStarted, nil)
	loaded, err := runner.loader.Load(ctx, &entry.LoadRequest{ProviderID: strings.TrimSpace(req.ProviderID)})
	if err != nil {
		runner.emitProgress(ctx, req, st, model.ProgressStageEntry, progressStatus(ctx, err), stageStarted, recordError(CodeProviderLoadFailed, "load provider", err))
		return nil, wrapError(CodeProviderLoadFailed, "load provider", err)
	}
	if loaded == nil {
		err := errors.New("load provider result is required")
		runner.emitProgress(ctx, req, st, model.ProgressStageEntry, model.ProgressStatusFailed, stageStarted, recordError(CodeProviderLoadFailed, "load provider", err))
		return nil, newError(CodeProviderLoadFailed, err.Error())
	}
	provider := loaded.Provider
	st.Provider = &provider
	runner.emitProgress(ctx, req, st, model.ProgressStageEntry, model.ProgressStatusSucceeded, stageStarted, nil)

	started, err := runner.tracer.Start(ctx, &tracelog.Request{
		TraceID:     strings.TrimSpace(req.TraceID),
		Hint:        req.Hint,
		ProviderIDs: st.providerIDs(),
	})
	if err != nil {
		return nil, wrapError(CodeTraceWriteFailed, "start trace", err)
	}
	if started == nil {
		return nil, newError(CodeTraceWriteFailed, "start trace result is required")
	}
	st.TraceID = started.Trace.ID
	st.StartedAt = started.Trace.StartedAt

	stageStarted = time.Now().UTC()
	runner.emitProgress(ctx, req, st, model.ProgressStageCatalog, model.ProgressStatusStarted, stageStarted, nil)
	if err := ctx.Err(); err != nil {
		runner.emitProgress(ctx, req, st, model.ProgressStageCatalog, model.ProgressStatusCanceled, stageStarted, recordError(CodeCatalogLoadFailed, "load capability catalog", err))
		return runner.terminate(ctx, req, st, CodeCatalogLoadFailed, "load capability catalog", err)
	}
	catalog, err := runner.catalog.ListCapabilities()
	if err != nil {
		runner.emitProgress(ctx, req, st, model.ProgressStageCatalog, model.ProgressStatusFailed, stageStarted, recordError(CodeCatalogLoadFailed, "load capability catalog", err))
		return runner.terminate(ctx, req, st, CodeCatalogLoadFailed, "load capability catalog", err)
	}
	st.Catalog = catalog
	runner.emitProgress(ctx, req, st, model.ProgressStageCatalog, model.ProgressStatusSucceeded, stageStarted, nil)

	stageStarted = time.Now().UTC()
	runner.emitProgress(ctx, req, st, model.ProgressStageObserve, model.ProgressStatusStarted, stageStarted, nil)
	observed, err := runner.observer.Observe(ctx, &observe.Request{Provider: st.Provider})
	if observed != nil {
		st.Observations = observed.Observations
	}
	if err != nil {
		runner.emitProgress(ctx, req, st, model.ProgressStageObserve, progressStatus(ctx, err), stageStarted, recordError(CodeObserveFailed, "observe provider", err))
		return runner.terminate(ctx, req, st, CodeObserveFailed, "observe provider", err)
	}
	if observed == nil {
		err := errors.New("observe result is required")
		runner.emitProgress(ctx, req, st, model.ProgressStageObserve, model.ProgressStatusFailed, stageStarted, recordError(CodeObserveFailed, "observe provider", err))
		return runner.terminate(ctx, req, st, CodeObserveFailed, "observe provider", err)
	}
	runner.emitProgress(ctx, req, st, model.ProgressStageObserve, model.ProgressStatusSucceeded, stageStarted, nil)

	stageStarted = time.Now().UTC()
	runner.emitProgress(ctx, req, st, model.ProgressStageProposal, model.ProgressStatusStarted, stageStarted, nil)
	proposalCtx := progress.WithHandler(ctx, progress.Handler(runner.onProgress))
	proposed, err := runner.proposer.Run(proposalCtx, &proposal.Request{
		Provider:     st.Provider,
		Observations: st.Observations,
		Catalog:      st.Catalog,
		Hint:         strings.TrimSpace(req.Hint),
		TraceID:      st.TraceID,
	})
	if proposed != nil {
		st.Proposal = proposed.Diagnostics
		st.Candidates = proposed.Candidates
		st.ProbePlans = proposed.ProbePlans
	}
	if err != nil {
		runner.emitProgress(ctx, req, st, model.ProgressStageProposal, progressStatus(ctx, err), stageStarted, recordError(CodeProposalFailed, "propose candidates", err))
		return runner.terminate(ctx, req, st, CodeProposalFailed, "propose candidates", err)
	}
	if proposed == nil {
		err := errors.New("proposal result is required")
		runner.emitProgress(ctx, req, st, model.ProgressStageProposal, model.ProgressStatusFailed, stageStarted, recordError(CodeProposalFailed, "propose candidates", err))
		return runner.terminate(ctx, req, st, CodeProposalFailed, "propose candidates", err)
	}
	runner.emitProgress(ctx, req, st, model.ProgressStageProposal, model.ProgressStatusSucceeded, stageStarted, nil)

	stageStarted = time.Now().UTC()
	runner.emitProgress(ctx, req, st, model.ProgressStageProbe, model.ProgressStatusStarted, stageStarted, nil)
	probed, err := runner.prober.Run(ctx, &probe.Request{
		Provider:   st.Provider,
		Candidates: st.Candidates,
		Plans:      st.ProbePlans,
		TraceID:    st.TraceID,
		WorkRoot:   strings.TrimSpace(req.WorkRoot),
	})
	if probed != nil {
		st.Probes = probed.Probes
	}
	if err != nil {
		runner.emitProgress(ctx, req, st, model.ProgressStageProbe, progressStatus(ctx, err), stageStarted, recordError(CodeProbeFailed, "probe candidates", err))
		return runner.terminate(ctx, req, st, CodeProbeFailed, "probe candidates", err)
	}
	if probed == nil {
		err := errors.New("probe result is required")
		runner.emitProgress(ctx, req, st, model.ProgressStageProbe, model.ProgressStatusFailed, stageStarted, recordError(CodeProbeFailed, "probe candidates", err))
		return runner.terminate(ctx, req, st, CodeProbeFailed, "probe candidates", err)
	}
	runner.emitProgress(ctx, req, st, model.ProgressStageProbe, model.ProgressStatusSucceeded, stageStarted, nil)

	stageStarted = time.Now().UTC()
	runner.emitProgress(ctx, req, st, model.ProgressStagePromote, model.ProgressStatusStarted, stageStarted, nil)
	promoted, err := runner.promoter.Run(ctx, &promote.Request{
		Candidates: st.Candidates,
		Probes:     st.Probes,
	})
	if promoted != nil {
		st.Promotions = promoted.Promotions
	}
	if err != nil {
		runner.emitProgress(ctx, req, st, model.ProgressStagePromote, progressStatus(ctx, err), stageStarted, recordError(CodePromotionFailed, "promote candidates", err))
		return runner.terminate(ctx, req, st, CodePromotionFailed, "promote candidates", err)
	}
	if promoted == nil {
		err := errors.New("promotion result is required")
		runner.emitProgress(ctx, req, st, model.ProgressStagePromote, model.ProgressStatusFailed, stageStarted, recordError(CodePromotionFailed, "promote candidates", err))
		return runner.terminate(ctx, req, st, CodePromotionFailed, "promote candidates", err)
	}
	runner.emitProgress(ctx, req, st, model.ProgressStagePromote, model.ProgressStatusSucceeded, stageStarted, nil)

	completed, err := runner.tracer.Complete(ctx, st.traceRequest(req, nil))
	if err != nil {
		return nil, wrapError(CodeTraceWriteFailed, "complete trace", err)
	}
	if completed == nil {
		return nil, newError(CodeTraceWriteFailed, "complete trace result is required")
	}
	return &Result{Trace: completed.Trace}, nil
}

func (runner *Runner) validate(req *Request) error {
	if runner == nil {
		return newError(CodeInvalidAcquisitionInput, "acquisition runner is required")
	}
	if runner.loader == nil || runner.catalog == nil || runner.observer == nil || runner.proposer == nil || runner.prober == nil || runner.promoter == nil || runner.tracer == nil {
		return newError(CodeInvalidAcquisitionInput, "all acquisition dependencies are required")
	}
	if req == nil {
		return newError(CodeInvalidAcquisitionInput, "acquisition request is required")
	}
	if strings.TrimSpace(req.ProviderID) == "" {
		return newError(CodeInvalidAcquisitionInput, "provider_id is required")
	}
	if strings.TrimSpace(req.WorkRoot) == "" {
		return newError(CodeInvalidAcquisitionInput, "work_root is required")
	}
	return nil
}

func (runner *Runner) terminate(ctx context.Context, req *Request, st *state, code string, message string, err error) (*Result, error) {
	stageErr := wrapError(code, message, err)
	traceReq := st.traceRequest(req, recordError(code, message, err))
	traceCtx := context.WithoutCancel(ctx)
	var written *tracelog.Result
	var traceErr error
	if isContextDone(ctx, err) {
		written, traceErr = runner.tracer.Cancel(traceCtx, traceReq)
	} else {
		written, traceErr = runner.tracer.Fail(traceCtx, traceReq)
	}
	if traceErr != nil {
		return nil, wrapError(CodeTraceWriteFailed, "write terminal trace", errors.Join(stageErr, traceErr))
	}
	if written == nil {
		return nil, wrapError(CodeTraceWriteFailed, "write terminal trace", stageErr)
	}
	return &Result{Trace: written.Trace}, stageErr
}

func isContextDone(ctx context.Context, err error) bool {
	if ctx != nil && ctx.Err() != nil {
		return true
	}
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func normalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
