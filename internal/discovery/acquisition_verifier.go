package discovery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/proposalflow"
	"github.com/spacehz-lab/cal/internal/runtime"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

type probeVerification struct {
	Provider       core.Provider
	Candidate      caltrace.Candidate
	Plan           proposalflow.ProbePlan
	CandidateIndex int
	WorkDir        string
	Now            time.Time
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
		return failedProbe(verification.CandidateIndex, nil, plan.Verifier, "probe_plan_failed", err, verification.Now), err
	}
	runner := runtime.NewRunner(runtime.DefaultRegistry())
	if _, err := runner.Execute(ctx, verification.Provider, verification.Candidate.Execution, plan.Inputs); err != nil {
		return failedProbe(verification.CandidateIndex, plan.Inputs, plan.Verifier, "execution_failed", err, verification.Now), err
	}
	evidence, _, err := runner.Verify(ctx, plan.Verifier, plan.Inputs)
	if err != nil {
		return failedProbe(verification.CandidateIndex, plan.Inputs, plan.Verifier, "verification_failed", err, verification.Now), err
	}
	return caltrace.Probe{
		CandidateIndex: verification.CandidateIndex,
		Passed:         true,
		Inputs:         copyProbeInputs(plan.Inputs),
		Verifier:       plan.Verifier,
		Evidence:       evidence,
		Reason:         plan.Verifier.ID + " verifier passed",
		CreatedAt:      verification.Now.Format(time.RFC3339Nano),
	}, nil
}

func failedProbe(candidateIndex int, inputs map[string]any, coreVerifier core.Verifier, code string, err error, now time.Time) caltrace.Probe {
	return caltrace.Probe{
		CandidateIndex: candidateIndex,
		Passed:         false,
		Inputs:         copyProbeInputs(inputs),
		Verifier:       coreVerifier,
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
