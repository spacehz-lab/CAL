package use

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Planner completes the input object for one selected binding.
type Planner struct {
	useID string
	now   time.Time
}

// NewPlanner builds a Use input planner.
func NewPlanner(useID string, now time.Time) Planner {
	return Planner{useID: useID, now: now}
}

// Plan merges caller inputs, LLM-extracted inputs, and CAL-generated target.
func (planner Planner) Plan(req Request, resolution Resolution) (map[string]any, *Error) {
	inputs := copyInputs(req.Inputs)
	for name, value := range resolution.Selection.inputsPatch {
		if hasInput(inputs, name) {
			return nil, &Error{Code: CodeInvalidLLMSelection, Message: fmt.Sprintf("inputs_patch key %q overwrites caller input", name)}
		}
		inputs[name] = value
	}
	if requiresTarget(resolution.Required) && !hasInput(inputs, "target") {
		target, err := planner.targetPath()
		if err != nil {
			return nil, &Error{Code: CodeArtifactPathFailed, Message: err.Error()}
		}
		inputs["target"] = target
	}
	if missing := missingInputs(resolution.Required, inputs); len(missing) > 0 {
		return nil, &Error{Code: CodeMissingInputs, Message: fmt.Sprintf("missing required inputs: %s", strings.Join(missing, ", "))}
	}
	return inputs, nil
}

func (planner Planner) targetPath() (string, error) {
	day := planner.now.UTC().Format("2006-01-02")
	dir := filepath.Join(os.TempDir(), "cal", "artifacts", day)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create temporary artifact directory: %w", err)
	}
	return filepath.Join(dir, planner.useID+".out"), nil
}

func copyInputs(inputs map[string]any) map[string]any {
	copied := make(map[string]any, len(inputs))
	for name, value := range inputs {
		copied[name] = value
	}
	return copied
}

func hasInput(inputs map[string]any, name string) bool {
	value, ok := inputs[name]
	return ok && value != nil && value != ""
}

func requiresTarget(required []string) bool {
	for _, name := range required {
		if name == "target" {
			return true
		}
	}
	return false
}
