package discovery

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/proposalflow"
	"github.com/spacehz-lab/cal/internal/runtime"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

const (
	contractEvidenceID   = "contract_probe_not_executed"
	contractEvidenceType = "contract"
	contractReason       = "contract verification accepted"
	l0Reason             = "L0 verification not promoted"
	defaultProbeTimeout  = 30 * time.Second
)

type probeVerification struct {
	Provider       core.Provider
	Candidate      caltrace.Candidate
	Plan           proposalflow.ProbePlan
	CandidateIndex int
	WorkDir        string
	Now            time.Time
	Timeout        time.Duration
}

func verifyProbe(ctx context.Context, verification probeVerification) (caltrace.Probe, error) {
	if verification.WorkDir == "" {
		return caltrace.Probe{}, fmt.Errorf("probe work directory is required")
	}
	if err := os.MkdirAll(verification.WorkDir, 0o755); err != nil {
		return caltrace.Probe{}, fmt.Errorf("create probe directory: %w", err)
	}
	workDir := filepath.Clean(verification.WorkDir)

	plan, err := proposalflow.MaterializeProbePlan(workDir, verification.Plan)
	if err != nil {
		return failedProbe(verification.CandidateIndex, nil, plan.Verify, "probe_plan_failed", err, verification.Now), err
	}
	if err := core.ValidateVerifySpec(plan.Verify); err != nil {
		return failedProbe(verification.CandidateIndex, plan.Inputs, plan.Verify, "verification_plan_invalid", err, verification.Now), err
	}
	if plan.Verify.Level == core.VerifyLevelL0 {
		return caltrace.Probe{
			CandidateIndex: verification.CandidateIndex,
			Passed:         false,
			Inputs:         copyProbeInputs(plan.Inputs),
			Verify:         plan.Verify,
			Reason:         l0Reason,
			CreatedAt:      verification.Now.Format(time.RFC3339Nano),
		}, nil
	}
	if plan.Verify.Method == core.VerifyMethodContract {
		return contractProbe(verification.CandidateIndex, plan.Inputs, plan.Verify, verification.Now), nil
	}
	runner := runtime.NewRunner(runtime.DefaultRegistry())
	executeCtx, cancel := context.WithTimeout(ctx, verification.timeout())
	defer cancel()
	result, err := runner.Execute(executeCtx, verification.Provider, verification.Candidate.Execution, plan.Inputs)
	if err != nil {
		if errors.Is(executeCtx.Err(), context.DeadlineExceeded) {
			return failedProbe(verification.CandidateIndex, plan.Inputs, plan.Verify, "execution_timeout", executeCtx.Err(), verification.Now), err
		}
		return failedProbe(verification.CandidateIndex, plan.Inputs, plan.Verify, "execution_failed", err, verification.Now), err
	}
	evidence, _, err := runner.Verify(ctx, plan.Verify, plan.Inputs, result)
	if err != nil {
		return failedProbe(verification.CandidateIndex, plan.Inputs, plan.Verify, "verification_failed", err, verification.Now), err
	}
	return caltrace.Probe{
		CandidateIndex: verification.CandidateIndex,
		Passed:         true,
		Inputs:         copyProbeInputs(plan.Inputs),
		Verify:         plan.Verify,
		Evidence:       evidence,
		Reason:         executeReason(plan.Verify),
		CreatedAt:      verification.Now.Format(time.RFC3339Nano),
	}, nil
}

func (verification probeVerification) timeout() time.Duration {
	if verification.Timeout > 0 {
		return verification.Timeout
	}
	return defaultProbeTimeout
}

func contractProbe(candidateIndex int, inputs map[string]any, verify core.VerifySpec, now time.Time) caltrace.Probe {
	return caltrace.Probe{
		CandidateIndex: candidateIndex,
		Passed:         true,
		Inputs:         copyProbeInputs(inputs),
		Verify:         verify,
		Evidence: []core.EvidenceRef{{
			ID:   contractEvidenceID,
			Type: contractEvidenceType,
			Content: map[string]any{
				"level":  verify.Level,
				"method": verify.Method,
				"reason": contractReason,
			},
		}},
		Reason:    contractReason,
		CreatedAt: now.Format(time.RFC3339Nano),
	}
}

func executeReason(verify core.VerifySpec) string {
	return string(verify.Level) + " verify checks passed"
}

func failedProbe(candidateIndex int, inputs map[string]any, verify core.VerifySpec, code string, err error, now time.Time) caltrace.Probe {
	return caltrace.Probe{
		CandidateIndex: candidateIndex,
		Passed:         false,
		Inputs:         copyProbeInputs(inputs),
		Verify:         verify,
		Reason:         code,
		Error: &core.RecordError{
			Code:    code,
			Message: err.Error(),
		},
		CreatedAt: now.Format(time.RFC3339Nano),
	}
}

func copyProbeInputs(inputs map[string]any) map[string]any {
	if len(inputs) == 0 {
		return nil
	}
	copied := make(map[string]any, len(inputs))
	for key, value := range inputs {
		copied[key] = value
	}
	return copied
}
