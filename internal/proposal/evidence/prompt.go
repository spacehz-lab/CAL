package evidence

import (
	"encoding/json"
	"sort"

	"github.com/spacehz-lab/cal/internal/llm"
	"github.com/spacehz-lab/cal/internal/model"
)

const promptKeyVerifyPredicateRules = "verify_predicate_rules"

const systemPrompt = `Return only JSON. For one candidate and probe material, plan how CAL should collect deterministic verification evidence.

Goal:
Stage 4 is Evidence. It only proposes verify method and checks so Proposal can derive the final verify level locally, then later Verification can execute the candidate and evaluate built-in checks or record weak contract evidence when execution is unsafe.

Response shape:
{"verify":{"method":"execute|contract","checks":[{"subject":{"type":"file|stdout|stderr|exit_code","input":"file input only"},"predicate":"equals|not_equals|exists|non_empty|format|contains|contains_any|regex|bytes_equal_transform|hash_line_matches","params":{}}]}}.

Boundary:
- Do not execute the candidate.
- Do not claim pass/fail or decide promotion.
- Do not write code, scripts, verifier packages, or natural-language oracles.
- Use only subjects, predicates, and params allowed by verify_subject_rules.
- For subject.type="file", subject.input must be one of available_file_inputs.
- Do not output verify.level. CAL derives level locally from method and checks.
- Output only the JSON object.

Internal decision process:
Before writing JSON, decide the VerifySpec internally in this order:
1. If the probe is safe, short, local, reads only probe fixtures, and writes only declared probe outputs inside the probe workdir, use method="execute".
2. If execution would install, remove, update, upgrade, clean, link, unlink, tap, untap, edit, start services, require network, require interaction, or change external state, use method="contract".
3. For contract, checks are advisory and are not executed.
4. For execute, choose the strongest built-in deterministic checks supported by probe_material and observations.
5. Use checks that prove the result, not the capability name or candidate description.
6. Do not use fixture-only sample content as durable expected output unless observations explicitly say the command always emits that literal.

Check rules:
- CAL executes the candidate and evaluates checks locally.
- Dry-run flags do not by themselves make package-manager update or upgrade commands safe.
- Contract verification does not execute checks and must not include pass/fail claims from execution.
- Checks must follow verify_subject_rules exactly.
- For subject.type="file", subject.input must be one of available_file_inputs.
- Do not invent subject types, file inputs, predicates, or params outside verify_subject_rules.
- Use verify_predicate_rules for predicate params. Emit a check only when all required_params for its predicate are present, and do not add params outside that predicate rule.
- A file subject input names a path input; it is not file content.
- Use file subjects for artifact path checks and file content checks.
- For generated file artifacts, prefer exists, non_empty, and format when they prove the output shape.
- Use file contains, contains_any, or regex only when observations explicitly guarantee stable literal output content.
- For stable literal output content, prefer contains over anchored full-file regex. Use anchored regex only when observations specify the exact whole file content including newline behavior.
- Fixture content is probe-only sample input. Do not use fixture content as durable expected output unless observations explicitly say the command always emits that literal.
- Use stdout, stderr, and exit_code subjects only for process result checks.`

func prompt(req *Request) *llm.Request {
	payload, _ := json.Marshal(map[string]any{
		"provider":                    req.Provider,
		"candidate_index":             req.CandidateIndex,
		"candidate":                   req.Candidate,
		"probe_material":              req.Material,
		"observations":                req.Observations,
		"verify_subject_rules":        verifySubjectRules(),
		promptKeyVerifyPredicateRules: verifyPredicateRules(),
		"available_file_inputs":       availableFileInputs(req.Material),
	})
	return &llm.Request{System: systemPrompt, User: string(payload), JSON: true}
}

func availableFileInputs(material Material) []string {
	inputs := map[string]struct{}{}
	for input := range material.Inputs {
		if input != "" {
			inputs[input] = struct{}{}
		}
	}
	for _, fixture := range material.Fixtures {
		if fixture.Input != "" {
			inputs[fixture.Input] = struct{}{}
		}
	}
	names := make([]string, 0, len(inputs))
	for input := range inputs {
		names = append(names, input)
	}
	sort.Strings(names)
	return names
}

func verifySubjectRules() []model.VerifySubjectRule {
	return []model.VerifySubjectRule{
		{
			Type:          model.VerifySubjectFile,
			RequiresInput: true,
			AllowedPredicates: []model.VerifyPredicate{
				model.VerifyPredicateExists,
				model.VerifyPredicateNonEmpty,
				model.VerifyPredicateFormat,
				model.VerifyPredicateContains,
				model.VerifyPredicateContainsAny,
				model.VerifyPredicateRegex,
				model.VerifyPredicateBytesEqualTransform,
				model.VerifyPredicateHashLineMatches,
			},
		},
		{
			Type: model.VerifySubjectStdout,
			AllowedPredicates: []model.VerifyPredicate{
				model.VerifyPredicateEquals,
				model.VerifyPredicateNotEquals,
				model.VerifyPredicateNonEmpty,
				model.VerifyPredicateContains,
				model.VerifyPredicateContainsAny,
				model.VerifyPredicateRegex,
				model.VerifyPredicateHashLineMatches,
			},
		},
		{
			Type: model.VerifySubjectStderr,
			AllowedPredicates: []model.VerifyPredicate{
				model.VerifyPredicateEquals,
				model.VerifyPredicateNotEquals,
				model.VerifyPredicateNonEmpty,
				model.VerifyPredicateContains,
				model.VerifyPredicateContainsAny,
				model.VerifyPredicateRegex,
				model.VerifyPredicateHashLineMatches,
			},
		},
		{
			Type: model.VerifySubjectExitCode,
			AllowedPredicates: []model.VerifyPredicate{
				model.VerifyPredicateEquals,
				model.VerifyPredicateNotEquals,
			},
		},
	}
}
