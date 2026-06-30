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
	var draft evidenceDraftOutput
	if err := json.Unmarshal(content, &draft); err != nil {
		return evidenceOutput{}, content, fmt.Errorf("decode evidence stage: %w", err)
	}
	output := evidenceOutput{Verify: draft.Verify.verifySpec()}
	output.Verify = normalizeEvidenceVerify(req, candidate, material, output.Verify)
	if err := core.ValidateVerifySpec(output.Verify); err != nil {
		return evidenceOutput{}, content, fmt.Errorf("evidence verify spec: %w", err)
	}
	if err := validateEvidenceInputs(output.Verify, material); err != nil {
		return evidenceOutput{}, content, fmt.Errorf("evidence verify spec: %w", err)
	}
	return output, content, nil
}

func (draft evidenceDraftVerify) verifySpec() core.VerifySpec {
	return core.VerifySpec{
		Method: draft.Method,
		Checks: draft.Checks,
	}
}

func normalizeEvidenceVerify(req Request, candidate caltrace.Candidate, material probeMaterial, verify core.VerifySpec) core.VerifySpec {
	fixtures := fixtureContents(material.Fixtures)
	switch verify.Method {
	case core.VerifyMethodContract:
		verify.Level = core.VerifyLevelL1
		return verify
	case core.VerifyMethodExecute:
	default:
		return verify
	}
	checks := verify.Checks
	if len(fixtures) > 0 && len(verify.Checks) > 0 {
		stableText := stableEvidenceText(req, candidate)
		checks = make([]core.VerifyCheck, 0, len(verify.Checks))
		for _, check := range verify.Checks {
			normalized, keep := normalizeEvidenceCheck(check, fixtures, stableText)
			if keep {
				checks = append(checks, normalized)
				continue
			}
		}
	}
	verify.Checks = checks
	verify.Level = deriveExecuteEvidenceLevel(checks)
	return verify
}

func validateEvidenceInputs(verify core.VerifySpec, material probeMaterial) error {
	if verify.Method != core.VerifyMethodExecute {
		return nil
	}
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

func normalizeEvidenceCheck(check core.VerifyCheck, fixtures []string, stableText string) (core.VerifyCheck, bool) {
	switch check.Predicate {
	case core.VerifyPredicateContains:
		value := evidenceStringParam(check.Params, "value")
		if fixtureOnlyLiteral(value, fixtures, stableText) {
			return check, false
		}
		return check, true
	case core.VerifyPredicateContainsAny:
		values := evidenceStringListParam(check.Params, "values")
		kept := make([]string, 0, len(values))
		for _, value := range values {
			if !fixtureOnlyLiteral(value, fixtures, stableText) {
				kept = append(kept, value)
			}
		}
		if len(kept) == 0 {
			return check, false
		}
		if len(kept) != len(values) {
			check.Params = copyParams(check.Params)
			check.Params["values"] = kept
			return check, true
		}
		return check, true
	case core.VerifyPredicateRegex:
		pattern := evidenceStringParam(check.Params, "pattern")
		if fixtureOnlyLiteral(pattern, fixtures, stableText) {
			return check, false
		}
		return check, true
	default:
		return check, true
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

type evidenceStrength int

const (
	evidenceStrengthNone evidenceStrength = iota
	evidenceStrengthProcess
	evidenceStrengthArtifact
	evidenceStrengthSemantic
)

func deriveExecuteEvidenceLevel(checks []core.VerifyCheck) core.VerifyLevel {
	if len(checks) == 0 {
		return core.VerifyLevelL0
	}
	strength := evidenceStrengthProcess
	for _, check := range checks {
		checkStrength := evidenceCheckStrength(check)
		if checkStrength > strength {
			strength = checkStrength
		}
	}
	switch strength {
	case evidenceStrengthSemantic:
		return core.VerifyLevelL3
	case evidenceStrengthArtifact:
		return core.VerifyLevelL2
	case evidenceStrengthProcess:
		return core.VerifyLevelL1
	default:
		return core.VerifyLevelL0
	}
}

func evidenceCheckStrength(check core.VerifyCheck) evidenceStrength {
	switch check.Predicate {
	case core.VerifyPredicateBytesEqualTransform, core.VerifyPredicateHashLineMatches, core.VerifyPredicateContains, core.VerifyPredicateContainsAny:
		return evidenceStrengthSemantic
	case core.VerifyPredicateFormat, core.VerifyPredicateRegex, core.VerifyPredicateNonEmpty:
		return evidenceStrengthArtifact
	case core.VerifyPredicateExists, core.VerifyPredicateEquals, core.VerifyPredicateNotEquals:
		return evidenceStrengthProcess
	default:
		return evidenceStrengthNone
	}
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
