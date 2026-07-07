package probe

import (
	"context"
	"errors"
	"time"

	"github.com/spacehz-lab/cal/internal/model"
)

const (
	ReasonVerifyChecksPassed      = "verify_checks_passed"
	ReasonContractEvidence        = "contract_evidence_recorded"
	ReasonVerificationLevelL0     = "verification_level_l0"
	ReasonVerificationPlanInvalid = "verification_plan_invalid"
	ReasonExecutionFailed         = "execution_failed"
	ReasonExecutionTimeout        = "execution_timeout"
	ReasonVerificationFailed      = "verification_failed"
	ReasonProbeMaterializeFailed  = "probe_materialize_failed"

	contractEvidenceID   = "contract_probe_not_executed"
	contractEvidenceType = "contract"
)

func passedProbe(target *Target, plan *MaterializedPlan, evidence []model.EvidenceRef, now time.Time) model.Probe {
	return model.Probe{
		CandidateIndex: target.CandidateIndex,
		Passed:         true,
		Inputs:         copyInputs(plan.Inputs),
		Verify:         plan.Verify,
		Evidence:       evidence,
		Reason:         ReasonVerifyChecksPassed,
		CreatedAt:      now.Format(time.RFC3339Nano),
	}
}

func contractProbe(target *Target, plan *MaterializedPlan, now time.Time) model.Probe {
	return model.Probe{
		CandidateIndex: target.CandidateIndex,
		Passed:         true,
		Inputs:         copyInputs(plan.Inputs),
		Verify:         plan.Verify,
		Evidence: []model.EvidenceRef{{
			ID:   contractEvidenceID,
			Type: contractEvidenceType,
			Content: map[string]any{
				"level":  plan.Verify.Level,
				"method": plan.Verify.Method,
				"reason": ReasonContractEvidence,
			},
		}},
		Reason:    ReasonContractEvidence,
		CreatedAt: now.Format(time.RFC3339Nano),
	}
}

func l0Probe(target *Target, plan *MaterializedPlan, now time.Time) model.Probe {
	return model.Probe{
		CandidateIndex: target.CandidateIndex,
		Passed:         false,
		Inputs:         copyInputs(plan.Inputs),
		Verify:         plan.Verify,
		Reason:         ReasonVerificationLevelL0,
		CreatedAt:      now.Format(time.RFC3339Nano),
	}
}

func materializeFailedProbe(target *Target, verify model.VerifySpec, err error, now time.Time) model.Probe {
	return failedProbe(target.CandidateIndex, nil, verify, ReasonProbeMaterializeFailed, CodeProbeMaterializeFailed, err, now)
}

func invalidVerifyProbe(target *Target, plan *MaterializedPlan, err error, now time.Time) model.Probe {
	return failedProbe(target.CandidateIndex, plan.Inputs, plan.Verify, ReasonVerificationPlanInvalid, CodeVerificationPlanInvalid, err, now)
}

func executionFailedProbe(target *Target, plan *MaterializedPlan, err error, now time.Time) model.Probe {
	code := CodeExecutionFailed
	reason := ReasonExecutionFailed
	if errors.Is(err, context.DeadlineExceeded) {
		code = CodeExecutionTimeout
		reason = ReasonExecutionTimeout
	}
	return failedProbe(target.CandidateIndex, plan.Inputs, plan.Verify, reason, code, err, now)
}

func verificationFailedProbe(target *Target, plan *MaterializedPlan, err error, now time.Time) model.Probe {
	return failedProbe(target.CandidateIndex, plan.Inputs, plan.Verify, ReasonVerificationFailed, CodeVerificationFailed, err, now)
}

func failedProbe(candidateIndex int, inputs map[string]any, verify model.VerifySpec, reason string, code string, err error, now time.Time) model.Probe {
	message := ""
	if err != nil {
		message = err.Error()
	}
	return model.Probe{
		CandidateIndex: candidateIndex,
		Passed:         false,
		Inputs:         copyInputs(inputs),
		Verify:         verify,
		Reason:         reason,
		Error:          &model.RecordError{Code: code, Message: message},
		CreatedAt:      now.Format(time.RFC3339Nano),
	}
}

func copyInputs(inputs map[string]any) map[string]any {
	if len(inputs) == 0 {
		return nil
	}
	copied := make(map[string]any, len(inputs))
	for key, value := range inputs {
		copied[key] = value
	}
	return copied
}
