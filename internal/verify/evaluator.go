package verify

import (
	"context"
	"fmt"

	"github.com/spacehz-lab/cal/internal/core"
)

// Context provides the observable result values for one verification run.
type Context struct {
	Inputs   map[string]any
	Stdout   string
	Stderr   string
	ExitCode int
}

// Evaluate runs deterministic checks from one VerifySpec.
func Evaluate(ctx context.Context, spec core.VerifySpec, verifyContext Context) ([]core.EvidenceRef, map[string]any, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if err := core.ValidateVerifySpec(spec); err != nil {
		return nil, nil, err
	}
	if spec.Method != core.VerifyMethodExecute {
		return nil, nil, fmt.Errorf("verify method %s is not executable", spec.Method)
	}
	if spec.Level == core.VerifyLevelL0 {
		return nil, nil, fmt.Errorf("verify level L0 is not executable")
	}
	evidence := make([]core.EvidenceRef, 0, len(spec.Checks))
	outputs := map[string]any{}
	for index, check := range spec.Checks {
		item, output, err := evaluateCheck(check, verifyContext, index)
		if err != nil {
			return nil, nil, err
		}
		evidence = append(evidence, item)
		for key, value := range output {
			outputs[key] = value
		}
	}
	return evidence, outputs, nil
}

func evaluateCheck(check core.VerifyCheck, verifyContext Context, index int) (core.EvidenceRef, map[string]any, error) {
	subject, err := evaluateSubject(check.Subject, verifyContext)
	if err != nil {
		return core.EvidenceRef{}, nil, err
	}
	if err := evaluatePredicate(check, subject); err != nil {
		return core.EvidenceRef{}, nil, err
	}
	id := fmt.Sprintf("check_%d_%s_%s", index+1, subject.label, check.Predicate)
	return core.EvidenceRef{
		ID:   id,
		Type: string(check.Predicate),
		Content: map[string]any{
			"subject":   check.Subject,
			"predicate": check.Predicate,
		},
	}, subject.outputs, nil
}
