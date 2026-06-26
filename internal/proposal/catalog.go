package proposal

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"

	"github.com/spacehz-lab/cal/internal/core"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

const (
	defaultCapabilityLimit      = 30
	smallCapabilityCatalogLimit = 50
)

var tokenPattern = regexp.MustCompile(`[a-z0-9_]+`)

// SelectExistingCapabilityIDs returns bounded lookup-first capability id candidates.
func SelectExistingCapabilityIDs(observations []caltrace.Observation, capabilities []core.Capability, hint string, limit int) []string {
	ids := validCapabilityIDs(capabilities)
	if len(ids) == 0 {
		return nil
	}
	if limit <= 0 {
		limit = defaultCapabilityLimit
	}
	if len(ids) <= smallCapabilityCatalogLimit {
		return prioritizeHint(ids, hint, len(ids))
	}

	text := observationText(observations)
	queryTokens := tokenize(text)
	queryText := strings.ToLower(text)
	capabilitiesByID := make(map[string]core.Capability, len(capabilities))
	for _, capability := range capabilities {
		if core.ValidCapabilityID(capability.ID) {
			capabilitiesByID[capability.ID] = capability
		}
	}
	scored := make([]scoredCapability, 0, len(ids))
	for _, id := range ids {
		capability := capabilitiesByID[id]
		score := scoreCapability(capability, queryText, queryTokens, hint)
		if score > 0 {
			scored = append(scored, scoredCapability{id: capability.ID, score: score})
		}
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].id < scored[j].id
	})

	selected := make([]string, 0, min(limit, len(scored)))
	for _, candidate := range scored {
		if len(selected) >= limit {
			break
		}
		selected = append(selected, candidate.id)
	}
	return selected
}

type scoredCapability struct {
	id    string
	score int
}

func validCapabilityIDs(capabilities []core.Capability) []string {
	seen := make(map[string]struct{}, len(capabilities))
	ids := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		if !core.ValidCapabilityID(capability.ID) {
			continue
		}
		if _, ok := seen[capability.ID]; ok {
			continue
		}
		seen[capability.ID] = struct{}{}
		ids = append(ids, capability.ID)
	}
	sort.Strings(ids)
	return ids
}

func prioritizeHint(ids []string, hint string, limit int) []string {
	selected := make([]string, 0, min(limit, len(ids)))
	if hint != "" {
		for _, id := range ids {
			if id == hint {
				selected = append(selected, id)
				break
			}
		}
	}
	for _, id := range ids {
		if len(selected) >= limit {
			break
		}
		if id == hint {
			continue
		}
		selected = append(selected, id)
	}
	return selected
}

func scoreCapability(capability core.Capability, queryText string, queryTokens map[string]struct{}, hint string) int {
	if capability.ID == hint {
		return 1000
	}
	score := 0
	if strings.Contains(queryText, capability.ID) {
		score += 200
	}

	domain, operation := splitCapabilityID(capability.ID)
	if _, ok := queryTokens[domain]; ok {
		score += 5
	}
	operationTokens := splitIDTokens(operation)
	matchedOperationTokens := 0
	for _, token := range operationTokens {
		if _, ok := queryTokens[token]; ok {
			score += 20
			matchedOperationTokens++
		}
	}
	if len(operationTokens) > 0 && matchedOperationTokens == len(operationTokens) {
		score += 30
	}

	for token := range tokenize(capability.Description) {
		if _, ok := queryTokens[token]; ok {
			score += 3
		}
	}
	return score
}

func splitCapabilityID(id string) (string, string) {
	parts := strings.SplitN(id, ".", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func splitIDTokens(text string) []string {
	tokens := tokenPattern.FindAllString(strings.ToLower(strings.ReplaceAll(text, ".", "_")), -1)
	var out []string
	for _, token := range tokens {
		for _, part := range strings.Split(token, "_") {
			if part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}

func tokenize(text string) map[string]struct{} {
	tokens := make(map[string]struct{})
	for _, token := range splitIDTokens(text) {
		tokens[token] = struct{}{}
	}
	return tokens
}

func observationText(observations []caltrace.Observation) string {
	var builder strings.Builder
	for _, observation := range observations {
		builder.WriteString(" ")
		builder.WriteString(observation.Type)
		builder.WriteString(" ")
		builder.WriteString(observation.Source)
		if len(observation.Content) > 0 {
			content, _ := json.Marshal(observation.Content)
			builder.WriteString(" ")
			builder.Write(content)
		}
	}
	return builder.String()
}
