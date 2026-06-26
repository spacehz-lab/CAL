package eval

import (
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

func TestRecordsReuseSeparatesRunFailureFromVerifierFailure(t *testing.T) {
	metrics := records{
		runs: []core.Run{
			{ID: "run_verified_success", Status: core.RunStatusSucceeded, Verified: true, DurationMS: 20},
			{ID: "run_execution_failed", Status: core.RunStatusFailed, DurationMS: 40},
			{ID: "run_verified_failed", Status: core.RunStatusFailed, Verified: true},
		},
	}.reuse()

	if metrics.RunCount != 3 || metrics.RunSuccessCount != 1 || metrics.RunFailureCount != 2 {
		t.Fatalf("run counts = %#v, want one success and two failures", metrics)
	}
	if metrics.VerifiedRunCount != 2 || metrics.VerifierFailCount != 1 {
		t.Fatalf("verified counts = %#v, want one verifier failure from verified runs only", metrics)
	}
	if metrics.RunSuccessRate != 1.0/3.0 || metrics.VerifiedSuccessRate != 0.5 || metrics.VerifierFailureRate != 0.5 {
		t.Fatalf("reuse rates = %#v, want separate run and verifier rates", metrics)
	}
	if metrics.AvgRunDurationMS != 30 {
		t.Fatalf("avg duration = %d, want 30", metrics.AvgRunDurationMS)
	}
}
