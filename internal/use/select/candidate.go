package selector

import (
	"regexp"
	"sort"
	"strings"

	"github.com/spacehz-lab/cal/internal/execute"
	"github.com/spacehz-lab/cal/internal/model"
)

const (
	scoreIDToken      = 4
	scoreDescToken    = 2
	scoreSatisfied    = 1
	scoreProviderHint = 4
)

var tokenPattern = regexp.MustCompile(`[a-z0-9]+`)

type candidate struct {
	capability model.Capability
	binding    model.Binding
	required   []string
	missing    []string
	score      int
}

type candidateBuilder struct {
	req          *Request
	intentTokens map[string]struct{}
	minLevel     model.VerifyLevel
}

func newCandidateBuilder(req *Request) *candidateBuilder {
	minLevel := req.MinVerifyLevel
	if minLevel == "" {
		minLevel = model.VerifyLevelL2
	}
	return &candidateBuilder{
		req:          req,
		intentTokens: tokenize(req.Intent),
		minLevel:     minLevel,
	}
}

func (builder *candidateBuilder) Build() []candidate {
	var candidates []candidate
	for _, capability := range builder.req.Capabilities {
		for _, binding := range capability.Bindings {
			candidate, ok := builder.scoreBinding(capability, binding)
			if ok {
				candidates = append(candidates, candidate)
			}
		}
	}
	return candidates
}

func (builder *candidateBuilder) scoreBinding(capability model.Capability, binding model.Binding) (candidate, bool) {
	if binding.State != model.BindingStatePromoted {
		return candidate{}, false
	}
	if strings.TrimSpace(builder.req.ProviderID) != "" && binding.ProviderID != strings.TrimSpace(builder.req.ProviderID) {
		return candidate{}, false
	}
	if binding.Verify == nil || model.VerifyLevelRank(binding.Verify.Level) < model.VerifyLevelRank(builder.minLevel) {
		return candidate{}, false
	}
	required, err := execute.RequiredInputs(&binding.Execution)
	if err != nil {
		return candidate{}, false
	}
	score := builder.intentScore(capability)
	if score == 0 {
		return candidate{}, false
	}
	missing := missingInputs(required, builder.req.Inputs)
	score += scoreSatisfied * (len(required) - len(missing))
	if strings.TrimSpace(builder.req.ProviderID) != "" {
		score += scoreProviderHint
	}
	return candidate{capability: capability, binding: binding, required: required, missing: missing, score: score}, true
}

func (builder *candidateBuilder) intentScore(capability model.Capability) int {
	score := 0
	for token := range tokenize(capability.ID) {
		if _, ok := builder.intentTokens[token]; ok {
			score += scoreIDToken
		}
	}
	for token := range tokenize(capability.Description) {
		if _, ok := builder.intentTokens[token]; ok {
			score += scoreDescToken
		}
	}
	return score
}

func tokenize(text string) map[string]struct{} {
	tokens := map[string]struct{}{}
	text = strings.ToLower(text)
	for _, token := range tokenPattern.FindAllString(text, -1) {
		tokens[token] = struct{}{}
	}
	return tokens
}

func missingInputs(required []string, inputs map[string]any) []string {
	var missing []string
	for _, name := range required {
		value, ok := inputs[name]
		if !ok || value == nil || value == "" {
			missing = append(missing, name)
		}
	}
	return missing
}

func hasNonTargetMissing(candidate candidate) bool {
	for _, name := range candidate.missing {
		if name != "target" {
			return true
		}
	}
	return false
}

func sortCandidates(candidates []candidate) {
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		if candidates[i].capability.ID != candidates[j].capability.ID {
			return candidates[i].capability.ID < candidates[j].capability.ID
		}
		return candidates[i].binding.ID < candidates[j].binding.ID
	})
}

func shouldUseLLM(candidates []candidate) bool {
	if len(candidates) == 0 {
		return false
	}
	if hasNonTargetMissing(candidates[0]) {
		return true
	}
	return len(candidates) > 1 && candidates[0].score == candidates[1].score
}

func resultFromCandidate(source Source, candidate candidate, reason string, considered int, inputsPatch map[string]any) *Result {
	return &Result{
		Source:               source,
		CapabilityID:         candidate.capability.ID,
		BindingID:            candidate.binding.ID,
		ProviderID:           candidate.binding.ProviderID,
		RequiredInputs:       append([]string(nil), candidate.required...),
		InputsPatch:          inputsPatch,
		Reason:               reason,
		CandidatesConsidered: considered,
	}
}
