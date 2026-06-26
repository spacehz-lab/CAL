package discovery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/proposal"
	"github.com/spacehz-lab/cal/internal/runtime"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

type acquisitionVerifier struct {
	planner proposal.ProbePlanner
}

func newAcquisitionVerifier(planner proposal.ProbePlanner) acquisitionVerifier {
	return acquisitionVerifier{planner: planner}
}

func (verifier acquisitionVerifier) Verify(ctx context.Context, provider core.Provider, candidate caltrace.Candidate, candidateIndex int, workDir string, now time.Time) (caltrace.Probe, error) {
	if workDir == "" {
		return caltrace.Probe{}, fmt.Errorf("probe work directory is required")
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return caltrace.Probe{}, fmt.Errorf("create probe directory: %w", err)
	}
	workDir = filepath.Clean(workDir)

	plan, err := verifier.planner.Plan(ctx, proposal.ProbePlanRequest{
		Candidate: candidate,
		WorkDir:   workDir,
	})
	if err != nil {
		return verifier.failedProbe(candidateIndex, nil, core.Verifier{}, "probe_plan_failed", err, now), err
	}
	runner := runtime.NewRunner(runtime.DefaultRegistry())
	if _, err := runner.Execute(ctx, provider, candidate.Execution, plan.Inputs); err != nil {
		return verifier.failedProbe(candidateIndex, plan.Inputs, plan.Verifier, "execution_failed", err, now), err
	}
	evidence, _, err := runner.Verify(ctx, plan.Verifier, plan.Inputs)
	if err != nil {
		return verifier.failedProbe(candidateIndex, plan.Inputs, plan.Verifier, "verification_failed", err, now), err
	}
	return caltrace.Probe{
		CandidateIndex: candidateIndex,
		Passed:         true,
		Inputs:         copyProbeInputs(plan.Inputs),
		Verifier:       plan.Verifier,
		Evidence:       evidence,
		Reason:         plan.Verifier.ID + " verifier passed",
		CreatedAt:      now.Format(time.RFC3339Nano),
	}, nil
}

func (verifier acquisitionVerifier) failedProbe(candidateIndex int, inputs map[string]any, coreVerifier core.Verifier, code string, err error, now time.Time) caltrace.Probe {
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
