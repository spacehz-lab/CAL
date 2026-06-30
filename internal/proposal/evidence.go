package proposal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spacehz-lab/cal/internal/core"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func (proposer *LLMProposer) draftEvidence(ctx context.Context, req Request, candidateIndex int, candidate caltrace.Candidate, material probeMaterial) (evidenceOutput, []byte, error) {
	content, err := proposer.client.Complete(ctx, cliEvidencePrompt(req, candidateIndex, candidate, material))
	if err != nil {
		return evidenceOutput{}, nil, fmt.Errorf("evidence stage: %w", err)
	}
	var output evidenceOutput
	if err := json.Unmarshal(content, &output); err != nil {
		return evidenceOutput{}, content, fmt.Errorf("decode evidence stage: %w", err)
	}
	if err := core.ValidateVerifySpec(output.Verify); err != nil {
		return evidenceOutput{}, content, fmt.Errorf("evidence verify spec: %w", err)
	}
	if err := validateEvidenceInputs(output.Verify, material); err != nil {
		return evidenceOutput{}, content, fmt.Errorf("evidence verify spec: %w", err)
	}
	output.Verify = normalizeEvidenceVerify(req, candidate, material, output.Verify)
	if err := core.ValidateVerifySpec(output.Verify); err != nil {
		return evidenceOutput{}, content, fmt.Errorf("evidence verify spec: %w", err)
	}
	return output, content, nil
}

func normalizeEvidenceVerify(req Request, candidate caltrace.Candidate, material probeMaterial, verify core.VerifySpec) core.VerifySpec {
	fixtures := fixtureContents(material.Fixtures)
	if verify.Method != core.VerifyMethodExecute || len(fixtures) == 0 || len(verify.Checks) == 0 {
		return verify
	}
	stableText := stableEvidenceText(req, candidate)
	checks := make([]core.VerifyCheck, 0, len(verify.Checks))
	changed := false
	for _, check := range verify.Checks {
		normalized, keep, checkChanged := normalizeEvidenceCheck(check, fixtures, stableText)
		if checkChanged {
			changed = true
		}
		if keep {
			checks = append(checks, normalized)
			continue
		}
	}
	if !changed {
		return verify
	}
	verify.Checks = checks
	verify.Level = capVerifyLevel(verify.Level, verifyLevelForChecks(checks))
	return verify
}

func validateEvidenceInputs(verify core.VerifySpec, material probeMaterial) error {
	available := probeInputSet(material)
	for _, check := range verify.Checks {
		if check.Subject.Type != core.VerifySubjectFile {
			continue
		}
		if _, ok := available[check.Subject.Input]; !ok {
			return fmt.Errorf("file subject input %q is not available", check.Subject.Input)
		}
	}
	return nil
}

func normalizeEvidenceCheck(check core.VerifyCheck, fixtures []string, stableText string) (core.VerifyCheck, bool, bool) {
	switch check.Predicate {
	case core.VerifyPredicateContains:
		value := evidenceStringParam(check.Params, "value")
		if fixtureOnlyLiteral(value, fixtures, stableText) {
			return check, false, true
		}
		return check, true, false
	case core.VerifyPredicateContainsAny:
		values := evidenceStringListParam(check.Params, "values")
		kept := make([]string, 0, len(values))
		for _, value := range values {
			if !fixtureOnlyLiteral(value, fixtures, stableText) {
				kept = append(kept, value)
			}
		}
		if len(kept) == 0 {
			return check, false, true
		}
		if len(kept) != len(values) {
			check.Params = copyParams(check.Params)
			check.Params["values"] = kept
			return check, true, true
		}
		return check, true, false
	case core.VerifyPredicateRegex:
		pattern := evidenceStringParam(check.Params, "pattern")
		if fixtureOnlyLiteral(pattern, fixtures, stableText) {
			return check, false, true
		}
		return check, true, false
	default:
		return check, true, false
	}
}

func fixtureOnlyLiteral(value string, fixtures []string, stableText string) bool {
	if value == "" || strings.Contains(stableText, value) {
		return false
	}
	for _, fixture := range fixtures {
		if strings.Contains(fixture, value) {
			return true
		}
	}
	return false
}

func fixtureContents(fixtures []Fixture) []string {
	values := make([]string, 0, len(fixtures))
	for _, fixture := range fixtures {
		content := strings.TrimSpace(fixture.Content)
		if content != "" {
			values = append(values, content)
		}
	}
	return values
}

func stableEvidenceText(req Request, candidate caltrace.Candidate) string {
	parts := []string{candidate.Description, candidate.Source}
	for _, observation := range req.Observations {
		parts = append(parts, observation.Source)
		if content, err := json.Marshal(observation.Content); err == nil {
			parts = append(parts, string(content))
		}
	}
	return strings.Join(parts, "\n")
}

func verifyLevelForChecks(checks []core.VerifyCheck) core.VerifyLevel {
	if len(checks) == 0 {
		return core.VerifyLevelL0
	}
	level := core.VerifyLevelL1
	for _, check := range checks {
		switch check.Predicate {
		case core.VerifyPredicateBytesEqualTransform, core.VerifyPredicateHashLineMatches, core.VerifyPredicateContains, core.VerifyPredicateContainsAny, core.VerifyPredicateRegex:
			return core.VerifyLevelL3
		case core.VerifyPredicateFormat:
			level = core.VerifyLevelL2
		}
	}
	return level
}

func capVerifyLevel(level, max core.VerifyLevel) core.VerifyLevel {
	if core.VerifyLevelRank(level) > core.VerifyLevelRank(max) {
		return max
	}
	return level
}

func evidenceStringParam(params map[string]any, key string) string {
	if len(params) == 0 {
		return ""
	}
	value, ok := params[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func evidenceStringListParam(params map[string]any, key string) []string {
	value, ok := params[key]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(item); text != "" {
				values = append(values, text)
			}
		}
		return values
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(fmt.Sprint(item)); text != "" {
				values = append(values, text)
			}
		}
		return values
	default:
		return nil
	}
}

func copyParams(params map[string]any) map[string]any {
	copied := make(map[string]any, len(params))
	for key, value := range params {
		copied[key] = value
	}
	return copied
}
