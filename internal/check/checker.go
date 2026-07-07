package check

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/spacehz-lab/cal/internal/model"
)

// Checker owns deterministic VerifySpec rules and evaluation.
type Checker struct {
	predicates map[model.VerifyPredicate]predicate
}

// NewChecker builds the default deterministic verification checker.
func NewChecker() *Checker {
	checker := &Checker{predicates: map[model.VerifyPredicate]predicate{}}
	registerTextPredicates(checker)
	registerFilePredicates(checker)
	registerBytesPredicates(checker)
	registerHashPredicates(checker)
	return checker
}

// Rules returns the prompt-facing VerifySpec subject contract.
func (c *Checker) Rules() []model.VerifySubjectRule {
	rules := make([]model.VerifySubjectRule, 0, len(subjectOrder))
	for _, subject := range subjectOrder {
		rule := model.VerifySubjectRule{
			Type:          subject,
			RequiresInput: subject == model.VerifySubjectFile,
		}
		for _, predicateName := range predicateOrder {
			predicate, ok := c.predicates[predicateName]
			if !ok || !predicate.allowsSubject(subject) {
				continue
			}
			rule.AllowedPredicates = append(rule.AllowedPredicates, predicateName)
			for _, param := range predicate.params {
				if param.required {
					if rule.RequiredParams == nil {
						rule.RequiredParams = map[model.VerifyPredicate][]string{}
					}
					rule.RequiredParams[predicateName] = append(rule.RequiredParams[predicateName], param.name)
				}
				if rule.ParamRules == nil {
					rule.ParamRules = map[model.VerifyPredicate]map[string]model.VerifyParamRule{}
				}
				if rule.ParamRules[predicateName] == nil {
					rule.ParamRules[predicateName] = map[string]model.VerifyParamRule{}
				}
				rule.ParamRules[predicateName][param.name] = model.VerifyParamRule{
					Required:      param.required,
					AllowedValues: append([]string(nil), param.allowedValues...),
				}
			}
		}
		rules = append(rules, rule)
	}
	return rules
}

// Validate checks deterministic VerifySpec semantics.
func (c *Checker) Validate(spec *model.VerifySpec) error {
	if spec == nil {
		return fmt.Errorf("verify spec is required")
	}
	if !validVerifyLevel(spec.Level) {
		return fmt.Errorf("verify level %q is invalid", spec.Level)
	}
	if !validVerifyMethod(spec.Method) {
		return fmt.Errorf("verify method %q is invalid", spec.Method)
	}
	if spec.Method == model.VerifyMethodContract && model.VerifyLevelRank(spec.Level) > model.VerifyLevelRank(model.VerifyLevelL1) {
		return fmt.Errorf("contract verification cannot exceed L1")
	}
	if spec.Method == model.VerifyMethodExecute && spec.Level != model.VerifyLevelL0 && len(spec.Checks) == 0 {
		return fmt.Errorf("verify checks are required for execute method")
	}
	if spec.Method != model.VerifyMethodExecute {
		return nil
	}
	for _, check := range spec.Checks {
		if err := c.validateCheck(&check); err != nil {
			return err
		}
	}
	return nil
}

// Run evaluates executable VerifySpec checks.
func (c *Checker) Run(ctx context.Context, req *Request) (*Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, fmt.Errorf("check request is required")
	}
	if err := c.Validate(req.Spec); err != nil {
		return nil, err
	}
	if req.Spec.Method != model.VerifyMethodExecute {
		return nil, fmt.Errorf("verify method %s is not executable", req.Spec.Method)
	}
	if req.Spec.Level == model.VerifyLevelL0 {
		return nil, fmt.Errorf("verify level L0 is not executable")
	}
	result := &Result{
		Evidence: make([]model.EvidenceRef, 0, len(req.Spec.Checks)),
		Outputs:  map[string]any{},
	}
	for index, check := range req.Spec.Checks {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		item, outputs, err := c.runCheck(&check, req, index)
		if err != nil {
			return nil, err
		}
		result.Evidence = append(result.Evidence, item)
		for key, value := range outputs {
			result.Outputs[key] = value
		}
	}
	return result, nil
}

func (c *Checker) runCheck(check *model.VerifyCheck, req *Request, index int) (model.EvidenceRef, map[string]any, error) {
	subject, err := resolveSubject(&check.Subject, req)
	if err != nil {
		return model.EvidenceRef{}, nil, err
	}
	if err := c.runPredicate(check, subject); err != nil {
		return model.EvidenceRef{}, nil, err
	}
	id := fmt.Sprintf("check_%d_%s_%s", index+1, subject.label, check.Predicate)
	return model.EvidenceRef{
		ID:   id,
		Type: string(check.Predicate),
		Content: map[string]any{
			evidenceContentSubject:   check.Subject,
			evidenceContentPredicate: check.Predicate,
		},
	}, subject.outputs, nil
}

func (c *Checker) validateCheck(check *model.VerifyCheck) error {
	requiresInput, ok := subjectRequiresInput(check.Subject.Type)
	if !ok {
		return fmt.Errorf("verify subject type %q is invalid", check.Subject.Type)
	}
	if requiresInput && strings.TrimSpace(check.Subject.Input) == "" {
		return fmt.Errorf("verify subject %s requires input", check.Subject.Type)
	}
	if !requiresInput && strings.TrimSpace(check.Subject.Input) != "" {
		return fmt.Errorf("verify subject %s cannot include input", check.Subject.Type)
	}
	predicate, ok := c.predicates[check.Predicate]
	if !ok {
		return fmt.Errorf("verify predicate %q is not supported", check.Predicate)
	}
	if !predicate.allowsSubject(check.Subject.Type) {
		return fmt.Errorf("verify predicate %s is invalid for subject %s", check.Predicate, check.Subject.Type)
	}
	if err := validateParams(check, predicate.params); err != nil {
		return err
	}
	if check.Predicate == model.VerifyPredicateRegex {
		pattern := stringParam(check.Params, paramPattern)
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("verify regex pattern is invalid: %w", err)
		}
	}
	return nil
}

func Rules() []model.VerifySubjectRule {
	return defaultChecker.Rules()
}

func Validate(spec *model.VerifySpec) error {
	return defaultChecker.Validate(spec)
}

func Run(ctx context.Context, req *Request) (*Result, error) {
	return defaultChecker.Run(ctx, req)
}

func validVerifyLevel(level model.VerifyLevel) bool {
	switch level {
	case model.VerifyLevelL0, model.VerifyLevelL1, model.VerifyLevelL2, model.VerifyLevelL3:
		return true
	default:
		return false
	}
}

func validVerifyMethod(method model.VerifyMethod) bool {
	switch method {
	case model.VerifyMethodExecute, model.VerifyMethodContract:
		return true
	default:
		return false
	}
}

var defaultChecker = NewChecker()
