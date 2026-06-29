package eval

import "github.com/spacehz-lab/cal/internal/core"

func (records records) reuse() ReuseMetrics {
	var succeeded int
	var failed int
	var verified int
	var verifiedSucceeded int
	var verifiedFailed int
	var durationTotal int64
	var durationCount int64

	for _, run := range records.runs {
		switch run.Status {
		case core.RunStatusSucceeded:
			succeeded++
		case core.RunStatusFailed:
			failed++
		}
		verifyFailed := run.Error != nil && run.Error.Code == "verification_failed"
		if run.Verified || verifyFailed {
			verified++
			if run.Status == core.RunStatusSucceeded {
				verifiedSucceeded++
			}
			if run.Status == core.RunStatusFailed {
				verifiedFailed++
			}
		}
		if run.DurationMS > 0 {
			durationTotal += run.DurationMS
			durationCount++
		}
	}

	metrics := ReuseMetrics{
		RunCount:            len(records.runs),
		RunSuccessCount:     succeeded,
		RunFailureCount:     failed,
		VerifiedRunCount:    verified,
		VerifierFailCount:   verifiedFailed,
		AvgRunDurationMS:    averageDuration(durationTotal, durationCount),
		VerifiedSuccessRate: 0,
		VerifierFailureRate: 0,
	}
	if len(records.runs) > 0 {
		metrics.RunSuccessRate = float64(succeeded) / float64(len(records.runs))
	}
	if verified > 0 {
		metrics.VerifiedSuccessRate = float64(verifiedSucceeded) / float64(verified)
		metrics.VerifierFailureRate = float64(verifiedFailed) / float64(verified)
	}
	return metrics
}

func averageDuration(total, count int64) int64 {
	if count == 0 {
		return 0
	}
	return total / count
}
