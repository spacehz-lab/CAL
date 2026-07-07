package probe

import (
	"fmt"
	"strings"
)

const workdirTemplate = "{{workdir}}"

// Materialize resolves one target's probe plan into concrete workdir inputs.
func Materialize(target *Target) (*MaterializedPlan, error) {
	if target == nil || target.Plan == nil {
		return nil, newError(CodeInvalidProbeInput, "probe target plan is required")
	}
	inputs, err := materializeInputs(target.WorkDir, target.Plan.Inputs)
	if err != nil {
		return nil, err
	}
	for _, fixture := range target.Plan.Fixtures {
		path, err := writeFixture(target.WorkDir, fixture)
		if err != nil {
			return nil, err
		}
		inputs[fixture.Input] = path
	}
	return &MaterializedPlan{
		CandidateIndex: target.CandidateIndex,
		Inputs:         inputs,
		Verify:         target.Plan.Verify,
		WorkDir:        target.WorkDir,
	}, nil
}

func materializeInputs(workDir string, inputs map[string]any) (map[string]any, error) {
	materialized := make(map[string]any, len(inputs))
	for key, value := range inputs {
		text, ok := value.(string)
		if !ok {
			materialized[key] = value
			continue
		}
		rendered := strings.ReplaceAll(text, workdirTemplate, workDir)
		if strings.Contains(rendered, "{{") || strings.Contains(rendered, "}}") {
			return nil, fmt.Errorf("probe input %q has unresolved template", key)
		}
		materialized[key] = rendered
	}
	return materialized, nil
}
