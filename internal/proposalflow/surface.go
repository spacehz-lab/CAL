package proposalflow

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func (proposer *LLMProposer) draftSurface(ctx context.Context, req Request, prof profile) ([]surface, []byte, caltrace.ProposalStage, error) {
	content, err := proposer.client.Complete(ctx, cliSurfacePrompt(req, prof))
	if err != nil {
		return nil, nil, caltrace.ProposalStage{}, fmt.Errorf("surface stage: %w", err)
	}
	var output surfaceOutput
	if err := json.Unmarshal(content, &output); err != nil {
		return nil, content, caltrace.ProposalStage{}, fmt.Errorf("decode surface stage: %w", err)
	}
	items, stage, err := normalizeSurfaces(output.SurfaceItems, proposer.policy.Surface, prof)
	if err != nil {
		return nil, content, stage, err
	}
	if len(items) == 0 {
		return nil, content, stage, fmt.Errorf("surface stage returned no kept surface items")
	}
	return items, content, stage, nil
}

func normalizeSurfaces(input []surface, policy SurfacePolicy, prof profile) ([]surface, caltrace.ProposalStage, error) {
	allowedKinds, skipNames, skipPatterns, err := compileSurfacePolicy(policy)
	if err != nil {
		return nil, caltrace.ProposalStage{}, err
	}
	items := make([]surface, 0, len(input))
	stage := caltrace.ProposalStage{
		Name: caltrace.ProposalStageSurface,
		Summary: map[caltrace.ProposalSummaryKey]int{
			caltrace.ProposalSummaryRaw: len(input),
		},
	}
	seen := map[string]struct{}{}
	for _, item := range input {
		item = normalizeSurfaceItem(item)
		if item.ID == "" || item.Name == "" {
			item.Decision = caltrace.ProposalDecisionSkip
			stage.Items = append(stage.Items, surfaceTraceItem(item))
			stage.Summary[caltrace.ProposalSummarySkip]++
			continue
		}
		if item.Decision != caltrace.ProposalDecisionKeep && item.Decision != caltrace.ProposalDecisionDefer && item.Decision != caltrace.ProposalDecisionSkip {
			item.Decision = caltrace.ProposalDecisionSkip
		}
		if item.Decision == caltrace.ProposalDecisionKeep {
			nameKey := strings.ToLower(item.Name)
			key := item.Kind + "\x00" + nameKey
			_, kindAllowed := allowedKinds[item.Kind]
			_, skippedName := skipNames[nameKey]
			_, duplicate := seen[key]
			switch {
			case !kindAllowed:
				item.Decision = caltrace.ProposalDecisionSkip
			case skippedName:
				item.Decision = caltrace.ProposalDecisionSkip
			case matchesAny(skipPatterns, item.Name):
				item.Decision = caltrace.ProposalDecisionSkip
			case duplicate:
				item.Decision = caltrace.ProposalDecisionSkip
			case prof.maxSurfaceItems > 0 && len(items) >= prof.maxSurfaceItems:
				item.Decision = caltrace.ProposalDecisionDefer
			default:
				seen[key] = struct{}{}
				items = append(items, item)
			}
		}
		stage.Items = append(stage.Items, surfaceTraceItem(item))
		stage.Summary[summaryKeyForDecision(item.Decision)]++
	}
	stage.Summary[caltrace.ProposalSummarySelected] = len(items)
	return items, stage, nil
}

func normalizeSurfaceItem(item surface) surface {
	item.ID = strings.TrimSpace(item.ID)
	item.Kind = normalizePolicyToken(item.Kind)
	if item.Kind == "" {
		item.Kind = "command"
	}
	item.Name = strings.TrimSpace(item.Name)
	item.Description = strings.TrimSpace(item.Description)
	item.EvidenceSource = strings.TrimSpace(item.EvidenceSource)
	item.Decision = caltrace.ProposalDecision(strings.ToLower(strings.TrimSpace(string(item.Decision))))
	if item.Decision == "" {
		item.Decision = caltrace.ProposalDecisionKeep
	}
	return item
}

func compileSurfacePolicy(policy SurfacePolicy) (map[string]struct{}, map[string]struct{}, []*regexp.Regexp, error) {
	if err := validateSurfacePolicy(policy); err != nil {
		return nil, nil, nil, err
	}
	allowedKinds := map[string]struct{}{}
	for _, kind := range policy.AllowedKinds {
		allowedKinds[normalizePolicyToken(kind)] = struct{}{}
	}
	skipNames := map[string]struct{}{}
	for _, name := range policy.SkipNames {
		name = strings.ToLower(strings.TrimSpace(name))
		if name != "" {
			skipNames[name] = struct{}{}
		}
	}
	patterns := make([]*regexp.Regexp, 0, len(policy.SkipPatterns))
	for _, pattern := range policy.SkipPatterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("invalid proposal surface skip pattern %q: %w", pattern, err)
		}
		patterns = append(patterns, compiled)
	}
	return allowedKinds, skipNames, patterns, nil
}

func matchesAny(patterns []*regexp.Regexp, value string) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(value) {
			return true
		}
	}
	return false
}

func summaryKeyForDecision(decision caltrace.ProposalDecision) caltrace.ProposalSummaryKey {
	switch decision {
	case caltrace.ProposalDecisionKeep:
		return caltrace.ProposalSummaryKeep
	case caltrace.ProposalDecisionDefer:
		return caltrace.ProposalSummaryDefer
	default:
		return caltrace.ProposalSummarySkip
	}
}

func surfaceTraceItem(item surface) caltrace.ProposalItem {
	return caltrace.ProposalItem{
		ID:       item.ID,
		Kind:     item.Kind,
		Name:     item.Name,
		Decision: item.Decision,
	}
}
