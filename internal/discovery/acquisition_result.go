package discovery

import (
	"log/slog"
	"time"

	"github.com/spacehz-lab/cal/internal/core"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func (run *acquisitionRun) fail(stage string, codedErr CodedError) (JobResult, error) {
	trace := caltrace.Trace{
		ID:           run.traceID,
		StartedAt:    run.now.UTC().Format(time.RFC3339Nano),
		EndedAt:      time.Now().UTC().Format(time.RFC3339Nano),
		Status:       caltrace.StatusFailed,
		Hint:         run.opts.CapabilityID,
		ProviderIDs:  run.providerIDs(),
		Observations: run.observations,
		Candidates:   run.candidates,
		Probes:       run.probes,
		Error: &core.RecordError{
			Code:    codedErr.Code,
			Message: codedErr.Message,
		},
	}
	if err := run.store.PutTrace(trace); err != nil {
		run.logFailed(stage, codedErr, false)
		return JobResult{}, err
	}
	run.logFailed(stage, codedErr, true)
	return JobResult{}, codedErr
}

func (run *acquisitionRun) complete() (JobResult, error) {
	trace := caltrace.Trace{
		ID:           run.traceID,
		StartedAt:    run.now.Format(time.RFC3339Nano),
		EndedAt:      time.Now().UTC().Format(time.RFC3339Nano),
		Status:       caltrace.StatusCompleted,
		Hint:         run.opts.CapabilityID,
		ProviderIDs:  run.providerIDs(),
		Observations: run.observations,
		Candidates:   run.candidates,
		Probes:       run.probes,
		Promotions:   run.promotions,
	}
	if err := run.store.PutTrace(trace); err != nil {
		return JobResult{}, err
	}

	run.logCompleted(run.traceID)
	return JobResult{
		JobID:                newJobID(run.now),
		State:                JobStateSucceeded,
		Target:               Target{Type: TargetProvider, ProviderID: run.opts.ProviderID},
		CapabilitiesPromoted: run.countCapabilityCreations(),
		BindingsPromoted:     len(run.promotions),
		ProposalDurationMS:   run.proposalMS,
		TraceID:              run.traceID,
		Providers:            []core.Provider{run.provider},
	}, nil
}

func (run *acquisitionRun) providerIDs() []string {
	if run.provider.ID == "" {
		return nil
	}
	return []string{run.provider.ID}
}

func (run *acquisitionRun) countCapabilityCreations() int {
	count := 0
	for _, promotion := range run.promotions {
		if promotion.CapabilityAction == "created" {
			count++
		}
	}
	return count
}

func (run *acquisitionRun) logStarted() {
	slog.Info("discovery acquisition started",
		"provider_id", run.opts.ProviderID,
		"capability_hint", run.opts.CapabilityID,
	)
}

func (run *acquisitionRun) logProviderLoaded() {
	slog.Info("discovery acquisition provider loaded",
		"provider_id", run.provider.ID,
		"provider_kind", run.provider.Kind,
	)
}

func (run *acquisitionRun) logObserved() {
	slog.Info("discovery acquisition observed",
		"provider_id", run.provider.ID,
		"observation_count", len(run.observations),
	)
}

func (run *acquisitionRun) logProposed(catalogSize int) {
	slog.Info("discovery acquisition proposed",
		"provider_id", run.provider.ID,
		"catalog_size", catalogSize,
		"candidate_count", len(run.candidates),
	)
}

func (run *acquisitionRun) logVerified() {
	passed := 0
	for _, probe := range run.probes {
		if probe.Passed {
			passed++
		}
	}
	slog.Info("discovery acquisition verified",
		"provider_id", run.provider.ID,
		"probe_count", len(run.probes),
		"passed_probe_count", passed,
		"failed_probe_count", len(run.probes)-passed,
	)
}

func (run *acquisitionRun) logPromoted() {
	slog.Info("discovery acquisition promoted",
		"provider_id", run.provider.ID,
		"promotion_count", len(run.promotions),
	)
}

func (run *acquisitionRun) logCompleted(traceID string) {
	slog.Info("discovery acquisition completed",
		"provider_id", run.provider.ID,
		"trace_id", traceID,
		"candidate_count", len(run.candidates),
		"probe_count", len(run.probes),
		"promotion_count", len(run.promotions),
	)
}

func (run *acquisitionRun) logFailed(stage string, err error, traceWritten bool) {
	code := "internal_error"
	if codedErr, ok := err.(CodedError); ok {
		code = codedErr.Code
	}
	slog.Warn("discovery acquisition failed",
		"provider_id", run.opts.ProviderID,
		"stage", stage,
		"code", code,
		"trace_written", traceWritten,
	)
}
