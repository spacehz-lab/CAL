package proposal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spacehz-lab/cal/internal/core"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func (proposer *LLMProposer) draftCapabilities(ctx context.Context, req Request, prof profile, surfaces []surface) ([]capabilityPlan, []byte, caltrace.ProposalStage, error) {
	content, err := proposer.client.Complete(ctx, cliCapabilityPrompt(req, prof, proposer.policy.Capability, surfaces))
	if err != nil {
		return nil, nil, caltrace.ProposalStage{}, fmt.Errorf("capability stage: %w", err)
	}
	var output capabilityOutput
	if err := json.Unmarshal(content, &output); err != nil {
		return nil, content, caltrace.ProposalStage{}, fmt.Errorf("decode capability stage: %w", err)
	}
	items, stage := normalizeCapabilities(output.Capabilities, proposer.policy.Capability, surfaces, req, prof)
	if len(items) == 0 {
		return nil, content, stage, fmt.Errorf("capability stage returned no capabilities")
	}
	return items, content, stage, nil
}

func normalizeCapabilities(input []capabilityPlan, policy CapabilityPolicy, surfaces []surface, req Request, prof profile) ([]capabilityPlan, caltrace.ProposalStage) {
	surfaceIDs := surfaceIDSet(surfaces)
	existingIDs := existingCapabilityIDSet(req)
	subjects := capabilityTermSet(policy.PreferredSubjects)
	operations := capabilityTermSet(policy.PreferredOperations)
	stage := caltrace.ProposalStage{
		Name: caltrace.ProposalStageCapability,
		Summary: map[caltrace.ProposalSummaryKey]int{
			caltrace.ProposalSummaryRaw: len(input),
		},
	}

	byID := map[string]int{}
	items := make([]capabilityPlan, 0, len(input))
	for index, item := range input {
		item = normalizeCapabilityPlan(item)
		traceItem := capabilityTraceItem(index, item)
		switch {
		case item.CapabilityID == "" || !core.ValidCapabilityID(item.CapabilityID):
			traceItem.Decision = caltrace.ProposalDecisionSkip
		case !validCapabilityParts(item.CapabilityID):
			traceItem.Decision = caltrace.ProposalDecisionSkip
		case req.DebugFilter != "" && item.CapabilityID != req.DebugFilter:
			traceItem.Decision = caltrace.ProposalDecisionSkip
		case !validCapabilitySources(item.SourceSurfaceIDs, surfaceIDs):
			traceItem.Decision = caltrace.ProposalDecisionSkip
		default:
			subject, operation, _ := strings.Cut(item.CapabilityID, ".")
			_, subjectInPolicy := subjects[subject]
			_, operationInPolicy := operations[operation]
			traceItem.Decision = caltrace.ProposalDecisionKeep
			if existing, ok := byID[item.CapabilityID]; ok {
				items[existing].SourceSurfaceIDs = mergeStrings(items[existing].SourceSurfaceIDs, item.SourceSurfaceIDs)
			} else if prof.maxCapabilities > 0 && len(items) >= prof.maxCapabilities {
				traceItem.Decision = caltrace.ProposalDecisionDefer
			} else {
				byID[item.CapabilityID] = len(items)
				items = append(items, item)
				if existingIDs[item.CapabilityID] {
					stage.Summary[caltrace.ProposalSummaryReused]++
				} else {
					stage.Summary[caltrace.ProposalSummaryCreated]++
				}
				if !subjectInPolicy || !operationInPolicy {
					stage.Summary[caltrace.ProposalSummaryOutOfPolicy]++
				}
			}
		}
		stage.Items = append(stage.Items, traceItem)
		stage.Summary[summaryKeyForDecision(traceItem.Decision)]++
	}
	stage.Summary[caltrace.ProposalSummarySelected] = len(items)
	return items, stage
}

func existingCapabilities(req Request, limit int) []existingCapabilityRef {
	items := make([]existingCapabilityRef, 0, len(req.Catalog))
	seen := map[string]struct{}{}
	if req.DebugFilter != "" {
		for _, capability := range req.Catalog {
			if capability.ID == req.DebugFilter && validExistingCapabilityID(capability.ID) {
				items = append(items, existingCapability(capability))
				seen[capability.ID] = struct{}{}
				break
			}
		}
	}
	for _, capability := range req.Catalog {
		if !validExistingCapabilityID(capability.ID) {
			continue
		}
		if _, ok := seen[capability.ID]; ok {
			continue
		}
		items = append(items, existingCapability(capability))
		seen[capability.ID] = struct{}{}
		if limit > 0 && len(items) >= limit {
			break
		}
	}
	return items
}

func existingCapability(capability core.Capability) existingCapabilityRef {
	return existingCapabilityRef{
		ID:          capability.ID,
		Description: strings.TrimSpace(capability.Description),
	}
}

func validExistingCapabilityID(id string) bool {
	return core.ValidCapabilityID(id) && validCapabilityParts(id)
}

func normalizeCapabilityPlan(item capabilityPlan) capabilityPlan {
	item.CapabilityID = strings.ToLower(strings.TrimSpace(item.CapabilityID))
	item.Description = strings.TrimSpace(item.Description)
	item.Confidence = strings.ToLower(strings.TrimSpace(item.Confidence))
	item.SourceSurfaceIDs = normalizeUniqueStrings(item.SourceSurfaceIDs)
	return item
}

func validCapabilityParts(id string) bool {
	subject, operation, ok := strings.Cut(id, ".")
	return ok && capabilityTermPattern.MatchString(subject) && capabilityTermPattern.MatchString(operation)
}

func validCapabilitySources(ids []string, surfaceIDs map[string]struct{}) bool {
	if len(ids) == 0 {
		return false
	}
	for _, id := range ids {
		if _, ok := surfaceIDs[id]; !ok {
			return false
		}
	}
	return true
}

func capabilityTraceItem(index int, item capabilityPlan) caltrace.ProposalItem {
	id := item.CapabilityID
	if id == "" {
		id = fmt.Sprintf("c%d", index+1)
	}
	return caltrace.ProposalItem{
		ID:   id,
		Kind: "capability",
		Name: item.CapabilityID,
	}
}

func surfaceIDSet(surfaces []surface) map[string]struct{} {
	ids := make(map[string]struct{}, len(surfaces))
	for _, surface := range surfaces {
		ids[surface.ID] = struct{}{}
	}
	return ids
}

func existingCapabilityIDSet(req Request) map[string]bool {
	ids := map[string]bool{}
	for _, capability := range req.Catalog {
		if capability.ID != "" {
			ids[capability.ID] = true
		}
	}
	return ids
}

func capabilityTermSet(terms []string) map[string]struct{} {
	set := make(map[string]struct{}, len(terms))
	for _, term := range terms {
		set[normalizePolicyToken(term)] = struct{}{}
	}
	return set
}

func normalizeUniqueStrings(values []string) []string {
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
	seen := map[string]struct{}{}
	merged := make([]string, 0, len(left)+len(right))
	for _, value := range append(left, right...) {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		merged = append(merged, value)
	}
	return merged
}
