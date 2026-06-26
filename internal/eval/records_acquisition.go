package eval

import (
	"sort"

	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

type acquisitionAggregator struct {
	metrics      AcquisitionMetrics
	byCapability map[string]*CapabilityAcquisitionMetrics
	bySource     map[string]*SourceAcquisitionMetrics
}

func (records records) acquisition() AcquisitionMetrics {
	aggregator := newAcquisitionAggregator()
	for _, record := range records.traces {
		aggregator.addTrace(record)
	}
	return aggregator.finish()
}

func newAcquisitionAggregator() *acquisitionAggregator {
	return &acquisitionAggregator{
		byCapability: map[string]*CapabilityAcquisitionMetrics{},
		bySource:     map[string]*SourceAcquisitionMetrics{},
	}
}

func (aggregator *acquisitionAggregator) addTrace(record caltrace.Trace) {
	if !isAcquisitionAttempt(record) {
		return
	}
	aggregator.metrics.AttemptCount++
	switch record.Status {
	case caltrace.StatusCompleted:
		aggregator.metrics.CompletedCount++
	case caltrace.StatusFailed:
		aggregator.metrics.FailedCount++
	}

	promotions := record.Promotions
	aggregator.metrics.PromotionCount += len(promotions)
	for _, promotion := range promotions {
		aggregator.addPromotionAction(promotion)
	}
	aggregator.metrics.CandidateCount += len(record.Candidates)
	aggregator.metrics.ProbeCount += len(record.Probes)
	aggregator.addTraceBuckets(record, promotions)
	aggregator.addProbeOutcomes(record.Probes)
}

func (aggregator *acquisitionAggregator) addPromotionAction(promotion caltrace.Promotion) {
	switch promotion.CapabilityAction {
	case "created":
		aggregator.metrics.CapabilityCreatedCount++
	case "reused":
		aggregator.metrics.CapabilityReusedCount++
	}
	switch promotion.BindingAction {
	case "created":
		aggregator.metrics.BindingCreatedCount++
	case "updated":
		aggregator.metrics.BindingUpdatedCount++
	}
}

func (aggregator *acquisitionAggregator) addTraceBuckets(record caltrace.Trace, promotions []caltrace.Promotion) {
	if len(record.Candidates) == 0 {
		aggregator.addTraceWithoutCandidates(record, promotions)
		return
	}

	promotionsByCandidate := map[int]int{}
	for _, promotion := range promotions {
		promotionsByCandidate[promotion.CandidateIndex]++
	}
	for index, candidate := range record.Candidates {
		aggregator.addCandidate(record, index, candidate, promotionsByCandidate[index])
	}
	for _, probe := range record.Probes {
		candidate, ok := candidateByProbe(record.Candidates, probe)
		if !ok {
			continue
		}
		capabilityMetrics := aggregator.capabilityBucket(candidate.CapabilityID)
		sourceMetrics := aggregator.sourceBucket(candidateSource(candidate))
		capabilityMetrics.Probes++
		sourceMetrics.Probes++
		countBucketProbe(probe, capabilityMetrics, sourceMetrics)
	}
}

func (aggregator *acquisitionAggregator) addTraceWithoutCandidates(record caltrace.Trace, promotions []caltrace.Promotion) {
	capabilityMetrics := aggregator.capabilityBucket(traceCapabilityID(record))
	sourceMetrics := aggregator.sourceBucket(traceSource(record))
	countTraceAttempt(record.Status, capabilityMetrics, sourceMetrics)
	capabilityMetrics.Probes += len(record.Probes)
	sourceMetrics.Probes += len(record.Probes)
	for _, promotion := range promotions {
		capabilityMetrics.Promotions++
		sourceMetrics.Promotions++
		if promotion.CapabilityID != capabilityMetrics.CapabilityID {
			aggregator.capabilityBucket(promotion.CapabilityID).Promotions++
		}
	}
	for _, probe := range record.Probes {
		countBucketProbe(probe, capabilityMetrics, sourceMetrics)
	}
}

func (aggregator *acquisitionAggregator) addCandidate(record caltrace.Trace, index int, candidate caltrace.Candidate, promotions int) {
	capabilityMetrics := aggregator.capabilityBucket(candidate.CapabilityID)
	sourceMetrics := aggregator.sourceBucket(candidateSource(candidate))
	capabilityMetrics.Attempts++
	sourceMetrics.Attempts++
	capabilityMetrics.Candidates++
	sourceMetrics.Candidates++
	if promotions > 0 {
		capabilityMetrics.Completed++
		sourceMetrics.Completed++
		capabilityMetrics.Promotions += promotions
		sourceMetrics.Promotions += promotions
		return
	}
	if record.Status == caltrace.StatusFailed || candidateHasFailedProbe(index, record.Probes) {
		capabilityMetrics.Failed++
		sourceMetrics.Failed++
	}
}

func (aggregator *acquisitionAggregator) addProbeOutcomes(probes []caltrace.Probe) {
	for _, probe := range probes {
		if probe.Passed {
			aggregator.metrics.ProbePassCount++
		} else {
			aggregator.metrics.ProbeFailCount++
		}
	}
}

func (aggregator *acquisitionAggregator) finish() AcquisitionMetrics {
	if aggregator.metrics.CandidateCount > 0 {
		aggregator.metrics.BindingPromotionRate = float64(aggregator.metrics.PromotionCount) / float64(aggregator.metrics.CandidateCount)
	}
	if aggregator.metrics.ProbeCount > 0 {
		aggregator.metrics.ProbeSuccessRate = float64(aggregator.metrics.ProbePassCount) / float64(aggregator.metrics.ProbeCount)
	}
	aggregator.metrics.ByCapability = aggregator.sortedCapabilityMetrics()
	if len(aggregator.metrics.ByCapability) == 0 {
		aggregator.metrics.ByCapability = nil
	}
	aggregator.metrics.BySource = aggregator.sortedSourceMetrics()
	if len(aggregator.metrics.BySource) == 0 {
		aggregator.metrics.BySource = nil
	}
	return aggregator.metrics
}

func (aggregator *acquisitionAggregator) capabilityBucket(capabilityID string) *CapabilityAcquisitionMetrics {
	metrics := aggregator.byCapability[capabilityID]
	if metrics == nil {
		metrics = &CapabilityAcquisitionMetrics{CapabilityID: capabilityID}
		aggregator.byCapability[capabilityID] = metrics
	}
	return metrics
}

func (aggregator *acquisitionAggregator) sourceBucket(source string) *SourceAcquisitionMetrics {
	metrics := aggregator.bySource[source]
	if metrics == nil {
		metrics = &SourceAcquisitionMetrics{Source: source}
		aggregator.bySource[source] = metrics
	}
	return metrics
}

func (aggregator *acquisitionAggregator) sortedCapabilityMetrics() []CapabilityAcquisitionMetrics {
	ids := make([]string, 0, len(aggregator.byCapability))
	for capabilityID := range aggregator.byCapability {
		ids = append(ids, capabilityID)
	}
	sort.Strings(ids)

	metrics := make([]CapabilityAcquisitionMetrics, 0, len(ids))
	for _, capabilityID := range ids {
		metrics = append(metrics, *aggregator.byCapability[capabilityID])
	}
	return metrics
}

func (aggregator *acquisitionAggregator) sortedSourceMetrics() []SourceAcquisitionMetrics {
	sources := make([]string, 0, len(aggregator.bySource))
	for source := range aggregator.bySource {
		sources = append(sources, source)
	}
	sort.Strings(sources)

	metrics := make([]SourceAcquisitionMetrics, 0, len(sources))
	for _, source := range sources {
		metrics = append(metrics, *aggregator.bySource[source])
	}
	return metrics
}

func countTraceAttempt(status caltrace.Status, capabilityMetrics *CapabilityAcquisitionMetrics, sourceMetrics *SourceAcquisitionMetrics) {
	capabilityMetrics.Attempts++
	sourceMetrics.Attempts++
	if status == caltrace.StatusCompleted {
		capabilityMetrics.Completed++
		sourceMetrics.Completed++
	}
	if status == caltrace.StatusFailed {
		capabilityMetrics.Failed++
		sourceMetrics.Failed++
	}
}

func countBucketProbe(probe caltrace.Probe, capabilityMetrics *CapabilityAcquisitionMetrics, sourceMetrics *SourceAcquisitionMetrics) {
	if probe.Passed {
		capabilityMetrics.ProbePasses++
		sourceMetrics.ProbePasses++
		return
	}
	capabilityMetrics.ProbeFailures++
	sourceMetrics.ProbeFailures++
}

func candidateByProbe(candidates []caltrace.Candidate, probe caltrace.Probe) (caltrace.Candidate, bool) {
	if probe.CandidateIndex < 0 || probe.CandidateIndex >= len(candidates) {
		return caltrace.Candidate{}, false
	}
	return candidates[probe.CandidateIndex], true
}

func candidateHasFailedProbe(candidateIndex int, probes []caltrace.Probe) bool {
	for _, probe := range probes {
		if probe.CandidateIndex == candidateIndex && !probe.Passed {
			return true
		}
	}
	return false
}

func traceSource(record caltrace.Trace) string {
	for _, candidate := range record.Candidates {
		return candidateSource(candidate)
	}
	return "unknown"
}

func candidateSource(candidate caltrace.Candidate) string {
	if candidate.Provenance != nil && candidate.Provenance.Source != "" {
		return candidate.Provenance.Source
	}
	if candidate.Source != "" {
		return candidate.Source
	}
	return "unknown"
}

func isAcquisitionAttempt(record caltrace.Trace) bool {
	return record.Hint != "" || len(record.Candidates) > 0 || len(record.Probes) > 0 || len(record.Promotions) > 0 || record.Error != nil
}

func traceCapabilityID(record caltrace.Trace) string {
	for _, promotion := range record.Promotions {
		if promotion.CapabilityID != "" {
			return promotion.CapabilityID
		}
	}
	for _, candidate := range record.Candidates {
		if candidate.CapabilityID != "" {
			return candidate.CapabilityID
		}
	}
	if record.Hint != "" {
		return record.Hint
	}
	return "unknown"
}
