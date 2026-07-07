package evidence

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spacehz-lab/cal/internal/model"
)

type output struct {
	Verify model.VerifySpec `json:"verify"`
}

// Parse decodes and locally filters evidence-stage JSON.
func Parse(raw string, req *Request) (model.VerifySpec, model.ProposalStage, error) {
	var out output
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return model.VerifySpec{}, model.ProposalStage{}, fmt.Errorf("decode evidence stage: %w", err)
	}
	verify, stage := normalize(out.Verify, req)
	if verify.Method == model.VerifyMethodExecute && verify.Level != model.VerifyLevelL0 && len(verify.Checks) == 0 {
		return verify, stage, fmt.Errorf("evidence stage returned no checks")
	}
	return verify, stage, nil
}

func normalize(verify model.VerifySpec, req *Request) (model.VerifySpec, model.ProposalStage) {
	if req == nil {
		req = &Request{}
	}
	if verify.Level == "" {
		verify.Level = model.VerifyLevelL1
	}
	if verify.Method == "" {
		verify.Method = model.VerifyMethodExecute
	}
	stage := newStage(model.ProposalStageEvidence, len(verify.Checks))
	selected := make([]model.VerifyCheck, 0, len(verify.Checks))
	inputs := inputSet(req.Material)
	predicateRules := predicateRuleMap(verifyPredicateRules())
	for index, check := range verify.Checks {
		trace := traceItem(index, check)
		paramReason := validatePredicateParams(check, predicateRules)
		switch {
		case check.Subject.Type == "":
			trace.Decision = model.ProposalDecisionSkip
			trace.Reason = "missing subject type"
		case check.Predicate == "":
			trace.Decision = model.ProposalDecisionSkip
			trace.Reason = "missing predicate"
		case check.Subject.Type == model.VerifySubjectFile && !inputs[check.Subject.Input]:
			trace.Decision = model.ProposalDecisionSkip
			trace.Reason = "unknown file input"
		case paramReason != "":
			trace.Decision = model.ProposalDecisionSkip
			trace.Reason = paramReason
		default:
			trace.Decision = model.ProposalDecisionKeep
			selected = append(selected, check)
		}
		stage.Items = append(stage.Items, trace)
		stage.Summary[summaryKey(trace.Decision)]++
	}
	verify.Checks = selected
	stage.Summary[model.ProposalSummarySelected] = len(selected)
	return verify, stage
}

func inputSet(material Material) map[string]bool {
	inputs := map[string]bool{}
	for key := range material.Inputs {
		inputs[key] = true
	}
	for _, fixture := range material.Fixtures {
		if fixture.Input != "" {
			inputs[fixture.Input] = true
		}
	}
	return inputs
}

func newStage(name model.ProposalStageName, raw int) model.ProposalStage {
	return model.ProposalStage{Name: name, Summary: map[model.ProposalSummaryKey]int{model.ProposalSummaryRaw: raw}}
}

func summaryKey(decision model.ProposalDecision) model.ProposalSummaryKey {
	switch decision {
	case model.ProposalDecisionKeep:
		return model.ProposalSummaryKeep
	case model.ProposalDecisionDefer:
		return model.ProposalSummaryDefer
	default:
		return model.ProposalSummarySkip
	}
}

func traceItem(index int, check model.VerifyCheck) model.ProposalItem {
	return model.ProposalItem{ID: fmt.Sprintf("e%d", index+1), Kind: string(check.Subject.Type), Name: string(check.Predicate)}
}

func newAttempt(started time.Time, raw string, err error, req *Request) model.ProposalAttempt {
	attempt := model.ProposalAttempt{Stage: model.ProposalStageEvidence, Status: model.ProposalAttemptSucceeded, DurationMS: time.Since(started).Milliseconds(), RawResponse: raw}
	if req != nil {
		attempt.CandidateIndex = &req.CandidateIndex
		if req.Candidate != nil {
			attempt.CapabilityID = req.Candidate.CapabilityID
		}
	}
	if err != nil {
		attempt.Status = model.ProposalAttemptFailed
		attempt.Error = &model.RecordError{Code: "proposal_stage_failed", Message: err.Error()}
	}
	return attempt
}
