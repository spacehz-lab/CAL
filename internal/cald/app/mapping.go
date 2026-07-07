package app

import (
	"context"
	"fmt"

	"github.com/spacehz-lab/cal/internal/acquisition"
	"github.com/spacehz-lab/cal/internal/contract"
	evalpkg "github.com/spacehz-lab/cal/internal/eval"
	"github.com/spacehz-lab/cal/internal/model"
	usepkg "github.com/spacehz-lab/cal/internal/use"
)

func (app *App) validate() error {
	if app == nil || app.store == nil || app.registry == nil || app.acquire == nil || app.run == nil || app.use == nil || app.eval == nil {
		return fmt.Errorf("cald app is not configured")
	}
	return nil
}

func ctxErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

func acquisitionResponse(result *acquisition.Result) *contract.AcquisitionResponse {
	if result == nil {
		return nil
	}
	trace := result.Trace
	capabilities, bindings := promotedCounts(trace.Promotions)
	return &contract.AcquisitionResponse{
		TraceID:              trace.ID,
		ProviderIDs:          append([]string(nil), trace.ProviderIDs...),
		CapabilitiesPromoted: capabilities,
		BindingsPromoted:     bindings,
		Trace:                &trace,
		Error:                trace.Error,
	}
}

func useResponse(result *usepkg.Result) *contract.UseResponse {
	if result == nil {
		return nil
	}
	return &contract.UseResponse{
		ID:         result.ID,
		Intent:     result.Intent,
		Selection:  useSelection(result.Selection),
		Run:        result.Run,
		Status:     result.Status,
		StartedAt:  result.StartedAt,
		FinishedAt: result.FinishedAt,
		DurationMS: result.DurationMS,
		Error:      result.Error,
	}
}

func evalResponse(result *evalpkg.Result) *contract.EvalResponse {
	if result == nil {
		return nil
	}
	return &contract.EvalResponse{
		Acquisition: acquisitionMetrics(result.Acquisition),
		Reuse:       reuseMetrics(result.Reuse),
		Capability:  capabilityMetrics(result.Capability),
	}
}

func useSelection(selection *usepkg.Selection) *contract.UseSelection {
	if selection == nil {
		return nil
	}
	return &contract.UseSelection{
		Source:               selection.Source,
		CapabilityID:         selection.CapabilityID,
		BindingID:            selection.BindingID,
		ProviderID:           selection.ProviderID,
		Reason:               selection.Reason,
		CandidatesConsidered: selection.CandidatesConsidered,
	}
}

func promotedCounts(promotions []model.Promotion) (int, int) {
	capabilities := map[string]struct{}{}
	bindings := map[string]struct{}{}
	for _, promotion := range promotions {
		if promotion.CapabilityID != "" {
			capabilities[promotion.CapabilityID] = struct{}{}
		}
		if promotion.BindingID != "" {
			bindings[promotion.BindingID] = struct{}{}
		}
	}
	return len(capabilities), len(bindings)
}

func acquisitionMetrics(metrics evalpkg.AcquisitionMetrics) contract.AcquisitionMetrics {
	return contract.AcquisitionMetrics{
		Traces:     countByStatus(metrics.Traces),
		Candidates: metrics.Candidates,
		Probes: contract.ProbeMetrics{
			Total:  metrics.Probes.Total,
			Passed: metrics.Probes.Passed,
			Failed: metrics.Probes.Failed,
		},
		Promotions: contract.PromotionMetrics{
			Total:        metrics.Promotions.Total,
			Capabilities: metrics.Promotions.Capabilities,
			Bindings:     metrics.Promotions.Bindings,
		},
		Errors: countByCode(metrics.Errors),
	}
}

func reuseMetrics(metrics evalpkg.ReuseMetrics) contract.ReuseMetrics {
	return contract.ReuseMetrics{
		Runs:         countByStatus(metrics.Runs),
		Verified:     metrics.Verified,
		ByProvider:   runSlices(metrics.ByProvider),
		ByCapability: runSlices(metrics.ByCapability),
		Errors:       countByCode(metrics.Errors),
	}
}

func capabilityMetrics(metrics evalpkg.CapabilityMetrics) contract.CapabilityMetrics {
	return contract.CapabilityMetrics{
		Capabilities:                metrics.Capabilities,
		Bindings:                    metrics.Bindings,
		PromotedBindings:            metrics.PromotedBindings,
		BindingsWithVerify:          metrics.BindingsWithVerify,
		CapabilitiesWithoutBindings: metrics.CapabilitiesWithoutBindings,
	}
}

func countByStatus(counts evalpkg.CountByStatus) contract.CountByStatus {
	return contract.CountByStatus{Total: counts.Total, ByName: stringIntMap(counts.ByName)}
}

func countByCode(counts evalpkg.CountByCode) contract.CountByCode {
	if len(counts) == 0 {
		return nil
	}
	copied := contract.CountByCode{}
	for code, count := range counts {
		copied[code] = count
	}
	return copied
}

func runSlices(values map[string]evalpkg.RunSlice) map[string]contract.RunSlice {
	if len(values) == 0 {
		return nil
	}
	copied := map[string]contract.RunSlice{}
	for key, value := range values {
		copied[key] = contract.RunSlice{
			Runs:     countByStatus(value.Runs),
			Verified: value.Verified,
		}
	}
	return copied
}

func stringIntMap(values map[string]int) map[string]int {
	if len(values) == 0 {
		return nil
	}
	copied := map[string]int{}
	for key, value := range values {
		copied[key] = value
	}
	return copied
}
