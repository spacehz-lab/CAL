package proposal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MaterializeProbePlan resolves workdir-relative inputs and fixture files.
func MaterializeProbePlan(workDir string, plan ProbePlan) (ProbePlan, error) {
	inputs, err := materializeProbeInputs(workDir, plan.Inputs)
	if err != nil {
		return plan, err
	}
	for _, fixture := range plan.Fixtures {
		path, err := materializeFixture(workDir, fixture)
		if err != nil {
			return plan, err
		}
		inputs[fixture.Input] = path
	}
	plan.Inputs = inputs
	return plan, nil
}

func materializeProbeInputs(workDir string, inputs map[string]any) (map[string]any, error) {
	if len(inputs) == 0 {
		return map[string]any{}, nil
	}
	materialized := make(map[string]any, len(inputs))
	for key, value := range inputs {
		text, ok := value.(string)
		if !ok {
			materialized[key] = value
			continue
		}
		rendered := strings.ReplaceAll(text, "{{workdir}}", workDir)
		if strings.Contains(rendered, "{{") || strings.Contains(rendered, "}}") {
			return nil, fmt.Errorf("probe input %q has unresolved template", key)
		}
		materialized[key] = rendered
	}
	return materialized, nil
}

func materializeFixture(workDir string, fixture Fixture) (string, error) {
	if fixture.Input == "" {
		return "", fmt.Errorf("probe fixture input is required")
	}
	if fixture.Filename == "" {
		return "", fmt.Errorf("probe fixture filename is required")
	}
	if filepath.IsAbs(fixture.Filename) {
		return "", fmt.Errorf("probe fixture filename must be relative")
	}
	clean := filepath.Clean(fixture.Filename)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("probe fixture filename escapes probe work directory")
	}
	path := filepath.Join(workDir, clean)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create probe fixture directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(fixture.Content), 0o644); err != nil {
		return "", fmt.Errorf("write probe fixture: %w", err)
	}
	return path, nil
}
