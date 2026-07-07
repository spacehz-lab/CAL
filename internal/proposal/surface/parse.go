package surface

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/proposal/policy"
)

type output struct {
	SurfaceItems []Item `json:"surface_items"`
}

func Parse(raw string, req *Request) ([]Item, model.ProposalStage, error) {
	var out output
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, model.ProposalStage{}, fmt.Errorf("decode surface stage: %w", err)
	}
	items, stage, err := normalize(out.SurfaceItems, req)
	if err != nil {
		return nil, stage, err
	}
	if len(items) == 0 {
		return nil, stage, fmt.Errorf("surface stage returned no kept surface items")
	}
	return items, stage, nil
}

func normalize(input []Item, req *Request) ([]Item, model.ProposalStage, error) {
	if req == nil {
		req = &Request{Policy: policy.Default().Surface}
	}
	if err := policy.ValidateSurface(req.Policy); err != nil {
		return nil, model.ProposalStage{}, err
	}
	allowedKinds := map[string]struct{}{}
	for _, kind := range req.Policy.AllowedKinds {
		allowedKinds[policy.NormalizeToken(kind)] = struct{}{}
	}
	skipNames := map[string]struct{}{}
	for _, name := range req.Policy.SkipNames {
		if name = strings.ToLower(strings.TrimSpace(name)); name != "" {
			skipNames[name] = struct{}{}
		}
	}
	patterns, err := compilePatterns(req.Policy.SkipPatterns)
	if err != nil {
		return nil, model.ProposalStage{}, err
	}

	stage := newStage(model.ProposalStageSurface, len(input))
	selected := make([]Item, 0, len(input))
	seen := map[string]struct{}{}
	for _, item := range input {
		item = normalizeItem(item)
		if item.ID == "" || item.Name == "" {
			item.Decision = model.ProposalDecisionSkip
		}
		if item.Decision == model.ProposalDecisionKeep {
			nameKey := strings.ToLower(item.Name)
			key := item.Kind + "\x00" + nameKey
			_, kindAllowed := allowedKinds[item.Kind]
			_, skippedName := skipNames[nameKey]
			_, duplicate := seen[key]
			switch {
			case !kindAllowed, skippedName, matchesAny(patterns, item.Name), duplicate:
				item.Decision = model.ProposalDecisionSkip
			case req.MaxItems > 0 && len(selected) >= req.MaxItems:
				item.Decision = model.ProposalDecisionDefer
			default:
				seen[key] = struct{}{}
				selected = append(selected, item)
			}
		}
		stage.Items = append(stage.Items, traceItem(item))
		stage.Summary[summaryKey(item.Decision)]++
	}
	stage.Summary[model.ProposalSummarySelected] = len(selected)
	return selected, stage, nil
}

func normalizeItem(item Item) Item {
	item.ID = strings.TrimSpace(item.ID)
	item.Kind = policy.NormalizeToken(item.Kind)
	if item.Kind == "" {
		item.Kind = "command"
	}
	item.Name = strings.TrimSpace(item.Name)
	item.Description = strings.TrimSpace(item.Description)
	item.Usage = strings.TrimSpace(item.Usage)
	item.EvidenceSource = strings.TrimSpace(item.EvidenceSource)
	item.Reason = strings.TrimSpace(item.Reason)
	item.Decision = model.ProposalDecision(strings.ToLower(strings.TrimSpace(string(item.Decision))))
	switch item.Decision {
	case model.ProposalDecisionKeep, model.ProposalDecisionDefer, model.ProposalDecisionSkip:
	default:
		item.Decision = model.ProposalDecisionKeep
	}
	return item
}

func compilePatterns(values []string) ([]*regexp.Regexp, error) {
	patterns := make([]*regexp.Regexp, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		pattern, err := regexp.Compile(value)
		if err != nil {
			return nil, fmt.Errorf("invalid proposal surface skip pattern %q: %w", value, err)
		}
		patterns = append(patterns, pattern)
	}
	return patterns, nil
}

func matchesAny(patterns []*regexp.Regexp, value string) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(value) {
			return true
		}
	}
	return false
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

func traceItem(item Item) model.ProposalItem {
	return model.ProposalItem{ID: item.ID, Kind: item.Kind, Name: item.Name, Decision: item.Decision, Reason: item.Reason}
}

func newAttempt(started time.Time, raw string, err error) model.ProposalAttempt {
	attempt := model.ProposalAttempt{Stage: model.ProposalStageSurface, Status: model.ProposalAttemptSucceeded, DurationMS: time.Since(started).Milliseconds(), RawResponse: raw}
	if err != nil {
		attempt.Status = model.ProposalAttemptFailed
		attempt.Error = &model.RecordError{Code: "proposal_stage_failed", Message: err.Error()}
	}
	return attempt
}
