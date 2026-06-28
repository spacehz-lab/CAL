package discovery

import (
	"context"
	"log/slog"
	"time"

	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/observe"
	"github.com/spacehz-lab/cal/internal/proposalflow"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

// Store is the persistence surface needed by discovery acquisition.
type Store interface {
	ListProviders() ([]core.Provider, error)
	ListCapabilities() ([]core.Capability, error)
	GetCapability(string) (core.Capability, bool, error)
	PutCapability(core.Capability) error
	PutTrace(caltrace.Trace) error
	PrepareTraceProbeDir(traceID string, candidateIndex int) (string, error)
}

// AcquisitionOptions identifies one provider acquisition. CapabilityID is an optional debug filter.
type AcquisitionOptions struct {
	ProviderID   string
	CapabilityID string
}

// AcquisitionRunner runs the current CLI-only provider acquisition slice.
type AcquisitionRunner struct {
	observer observe.Observer
	proposer proposalflow.Proposer
}

// NewAcquisitionRunner builds a provider acquisition runner with explicit dependencies.
func NewAcquisitionRunner(observer observe.Observer, proposer proposalflow.Proposer) AcquisitionRunner {
	return AcquisitionRunner{
		observer: observer,
		proposer: proposer,
	}
}

// Run executes provider Inference, Verification, Promotion, and Trace writing.
func (runner AcquisitionRunner) Run(ctx context.Context, store Store, opts AcquisitionOptions) (JobResult, error) {
	now := time.Now().UTC()
	if err := runner.validate(); err != nil {
		code := "internal_error"
		if codedErr, ok := err.(CodedError); ok {
			code = codedErr.Code
		}
		slog.Warn("discovery acquisition failed",
			"provider_id", opts.ProviderID,
			"stage", "configuration",
			"code", code,
			"trace_written", false,
		)
		return JobResult{}, err
	}

	run := newAcquisitionRun(runner, store, opts, now)
	return run.execute(ctx)
}

func (runner AcquisitionRunner) validate() error {
	if runner.observer == nil {
		return newCodedError(CodeObserverUnavailable, "discovery acquisition observer is not configured")
	}
	if runner.proposer == nil {
		return newCodedError(CodeProposerUnavailable, "discovery acquisition proposer is not configured")
	}
	return nil
}

type acquisitionRun struct {
	runner  AcquisitionRunner
	store   Store
	opts    AcquisitionOptions
	now     time.Time
	traceID string

	provider     core.Provider
	observations []caltrace.Observation
	candidates   []caltrace.Candidate
	probePlans   []proposalflow.ProbePlan
	probes       []caltrace.Probe
	promotions   []caltrace.Promotion
	proposalMS   int64
}

func newAcquisitionRun(runner AcquisitionRunner, store Store, opts AcquisitionOptions, now time.Time) *acquisitionRun {
	return &acquisitionRun{
		runner:  runner,
		store:   store,
		opts:    opts,
		now:     now,
		traceID: caltrace.NewID(now),
	}
}

func (run *acquisitionRun) execute(ctx context.Context) (JobResult, error) {
	run.logStarted()
	if err := run.loadProvider(); err != nil {
		run.logFailed("provider_load", err, false)
		return JobResult{}, err
	}
	run.logProviderLoaded()
	if err := run.observe(ctx); err != nil {
		return run.fail("observation", newCodedError(CodeObservationFailed, err.Error()))
	}
	run.logObserved()
	if codedErr := run.propose(ctx); codedErr.Code != "" {
		return run.fail("proposal", codedErr)
	}
	if codedErr := run.verify(ctx); codedErr.Code != "" {
		return run.fail("verification", codedErr)
	}
	run.logVerified()
	if codedErr := run.promote(); codedErr.Code != "" {
		return run.fail("promotion", codedErr)
	}
	run.logPromoted()
	return run.complete()
}

func (run *acquisitionRun) loadProvider() error {
	providers, err := run.store.ListProviders()
	if err != nil {
		return err
	}
	for _, provider := range providers {
		if provider.ID != run.opts.ProviderID {
			continue
		}
		if provider.Kind != core.ProviderKindCLI {
			return newCodedErrorf(CodeUnsupportedProviderKind, "provider kind %q is not supported for discovery acquisition", provider.Kind)
		}
		run.provider = provider
		return nil
	}
	return newCodedErrorf(CodeProviderNotFound, "provider %q was not found", run.opts.ProviderID)
}

func (run *acquisitionRun) observe(ctx context.Context) error {
	result, err := run.runner.observer.Observe(ctx, run.provider)
	if err != nil {
		return err
	}
	run.observations = make([]caltrace.Observation, 0, len(result.Observations))
	for _, observation := range result.Observations {
		run.observations = append(run.observations, caltrace.Observation{
			ProviderID: run.provider.ID,
			Type:       observation.Type,
			Source:     observation.Source,
			Content:    observation.Content,
			CreatedAt:  run.now.Format(time.RFC3339Nano),
		})
	}
	return nil
}

func (run *acquisitionRun) propose(ctx context.Context) CodedError {
	capabilities, err := run.store.ListCapabilities()
	if err != nil {
		return newCodedError(CodeCandidateProposalFailed, err.Error())
	}
	started := time.Now()
	result, err := run.runner.proposer.Propose(ctx, proposalflow.Request{
		Provider:     run.provider,
		Observations: run.observations,
		Catalog:      capabilities,
		DebugFilter:  run.opts.CapabilityID,
	})
	run.proposalMS = time.Since(started).Milliseconds()
	if err != nil {
		return newCodedError(CodeCandidateProposalFailed, err.Error())
	}
	if len(result.Candidates) == 0 {
		if run.opts.CapabilityID == "" {
			return newCodedError(CodeCandidateNotFound, "no candidate was proposed")
		}
		return newCodedErrorf(CodeCandidateNotFound, "no candidate was proposed for capability %q", run.opts.CapabilityID)
	}
	if len(result.ProbePlans) != len(result.Candidates) {
		return newCodedErrorf(CodeCandidateProposalFailed, "proposal returned %d candidates and %d probe plans", len(result.Candidates), len(result.ProbePlans))
	}
	run.candidates = result.Candidates
	run.probePlans = result.ProbePlans
	for index := range run.candidates {
		run.candidates[index].CreatedAt = run.now.Format(time.RFC3339Nano)
	}
	run.logProposed(len(capabilities))
	return CodedError{}
}

func (run *acquisitionRun) verify(ctx context.Context) CodedError {
	run.probes = make([]caltrace.Probe, 0, len(run.candidates))
	passed := 0
	var lastErr error
	for index, candidate := range run.candidates {
		workDir, err := run.store.PrepareTraceProbeDir(run.traceID, index)
		if err != nil {
			return newCodedError(CodeVerificationFailed, err.Error())
		}
		probe, err := verifyProbe(ctx, probeVerification{
			Provider:       run.provider,
			Candidate:      candidate,
			Plan:           run.probePlans[index],
			CandidateIndex: index,
			WorkDir:        workDir,
			Now:            run.now,
		})
		run.probes = append(run.probes, probe)
		if probe.Passed {
			passed++
		}
		if err != nil {
			lastErr = err
		}
	}
	if passed == 0 {
		if lastErr != nil {
			return newCodedError(CodeVerificationFailed, lastErr.Error())
		}
		return newCodedError(CodeVerificationFailed, "no candidate passed verification")
	}
	return CodedError{}
}

func (run *acquisitionRun) promote() CodedError {
	promoter := newAcquisitionPromoter(run.store, run.provider, run.now)
	for _, probe := range run.probes {
		if !probe.Passed {
			continue
		}
		if probe.CandidateIndex < 0 || probe.CandidateIndex >= len(run.candidates) {
			return newCodedErrorf(CodePromotionFailed, "probe candidate_index %d is out of range", probe.CandidateIndex)
		}
		candidate := run.candidates[probe.CandidateIndex]
		promotion, err := promoter.Promote(candidate, probe, probe.CandidateIndex)
		if err != nil {
			return newCodedError(CodePromotionFailed, err.Error())
		}
		run.promotions = append(run.promotions, promotion)
	}
	return CodedError{}
}
