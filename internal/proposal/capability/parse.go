package capability

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/proposal/policy"
)

type output struct {
	Capabilities []Plan `json:"capabilities"`
}

func Parse(raw string, req *Request) ([]Plan, model.ProposalStage, error) {
	var out output
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, model.ProposalStage{}, fmt.Errorf("decode capability stage: %w", err)
	}
	plans, stage := normalize(out.Capabilities, req)
	if len(plans) == 0 {
		return nil, stage, fmt.Errorf("capability stage returned no capabilities")
	}
	return plans, stage, nil
}

func normalize(input []Plan, req *Request) ([]Plan, model.ProposalStage) {
	if req == nil {
		req = &Request{Policy: policy.Default().Capability}
	}
	surfaceIDs := surfaceIDSet(req.Surfaces)
	existingIDs := existingCapabilityIDs(req.Catalog)
	stage := newStage(model.ProposalStageCapability, len(input))
	byID := map[string]int{}
	selected := make([]Plan, 0, len(input))
	for index, plan := range input {
		plan = normalizePlan(plan)
		trace := traceItem(index, plan)
		switch {
		case plan.CapabilityID == "" || !model.ValidCapabilityID(plan.CapabilityID):
			trace.Decision = model.ProposalDecisionSkip
		case !validSources(plan.SourceSurfaceIDs, surfaceIDs):
			trace.Decision = model.ProposalDecisionSkip
		default:
			trace.Decision = model.ProposalDecisionKeep
			if existing, ok := byID[plan.CapabilityID]; ok {
				selected[existing].SourceSurfaceIDs = mergeStrings(selected[existing].SourceSurfaceIDs, plan.SourceSurfaceIDs)
			} else if req.MaxPlans > 0 && len(selected) >= req.MaxPlans {
				trace.Decision = model.ProposalDecisionDefer
			} else {
				byID[plan.CapabilityID] = len(selected)
				selected = append(selected, plan)
				if existingIDs[plan.CapabilityID] {
					stage.Summary[model.ProposalSummaryReused]++
				} else {
					stage.Summary[model.ProposalSummaryCreated]++
				}
			}
		}
		stage.Items = append(stage.Items, trace)
		stage.Summary[summaryKey(trace.Decision)]++
	}
	stage.Summary[model.ProposalSummarySelected] = len(selected)
	return selected, stage
}

func normalizePlan(plan Plan) Plan {
	plan.CapabilityID = strings.ToLower(strings.TrimSpace(plan.CapabilityID))
	plan.Description = strings.TrimSpace(plan.Description)
	plan.Confidence = strings.ToLower(strings.TrimSpace(plan.Confidence))
	plan.SourceSurfaceIDs = normalizeStrings(plan.SourceSurfaceIDs)
	return plan
}

func surfaceIDSet(items []SurfaceItem) map[string]struct{} {
	ids := map[string]struct{}{}
	for _, item := range items {
		if item.ID != "" {
			ids[item.ID] = struct{}{}
		}
	}
	return ids
}

func existingCapabilityIDs(catalog []model.Capability) map[string]bool {
	ids := map[string]bool{}
	for _, capability := range catalog {
		if capability.ID != "" {
			ids[capability.ID] = true
		}
	}
	return ids
}

func validSources(ids []string, available map[string]struct{}) bool {
	if len(ids) == 0 {
		return false
	}
	for _, id := range ids {
		if _, ok := available[id]; !ok {
			return false
		}
	}
	return true
}

func normalizeStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func mergeStrings(left []string, right []string) []string {
	return normalizeStrings(append(left, right...))
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

func traceItem(index int, plan Plan) model.ProposalItem {
	id := plan.CapabilityID
	if id == "" {
		id = fmt.Sprintf("c%d", index+1)
	}
	return model.ProposalItem{ID: id, Kind: "capability", Name: plan.CapabilityID}
}

func newAttempt(started time.Time, raw string, err error) model.ProposalAttempt {
	attempt := model.ProposalAttempt{Stage: model.ProposalStageCapability, Status: model.ProposalAttemptSucceeded, DurationMS: time.Since(started).Milliseconds(), RawResponse: raw}
	if err != nil {
		attempt.Status = model.ProposalAttemptFailed
		attempt.Error = &model.RecordError{Code: "proposal_stage_failed", Message: err.Error()}
	}
	return attempt
}
