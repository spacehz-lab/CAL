package plan

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrNilRequest         = errors.New("use plan request is required")
	ErrInvalidInputsPatch = errors.New("invalid inputs patch")
	ErrMissingInputs      = errors.New("missing inputs")
)

// Request provides final input planning input.
type Request struct {
	UseID          string
	Now            time.Time
	Inputs         map[string]any
	RequiredInputs []string
	InputsPatch    map[string]any
}

// Result describes planned run inputs.
type Result struct {
	Inputs map[string]any
}

// Runner plans final inputs for one selected binding.
type Runner struct{}

// NewRunner creates an input planner.
func NewRunner() *Runner {
	return &Runner{}
}

// Run merges caller inputs, selector inputs, and generated local inputs.
func (runner *Runner) Run(req *Request) (*Result, error) {
	if req == nil {
		return nil, ErrNilRequest
	}
	inputs, err := mergeInputs(req.Inputs, req.InputsPatch)
	if err != nil {
		return nil, err
	}
	if requiresInput(req.RequiredInputs, InputTarget) && !hasInput(inputs, InputTarget) {
		target, err := TargetPath(req.UseID, req.Now)
		if err != nil {
			return nil, err
		}
		inputs[InputTarget] = target
	}
	if missing := missingInputs(req.RequiredInputs, inputs); len(missing) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrMissingInputs, strings.Join(missing, ", "))
	}
	return &Result{Inputs: inputs}, nil
}

func mergeInputs(inputs map[string]any, patch map[string]any) (map[string]any, error) {
	merged := copyInputs(inputs)
	for name, value := range patch {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("%w: empty key", ErrInvalidInputsPatch)
		}
		if name == InputTarget {
			return nil, fmt.Errorf("%w: target is generated locally", ErrInvalidInputsPatch)
		}
		if hasInput(merged, name) {
			return nil, fmt.Errorf("%w: %s overwrites caller input", ErrInvalidInputsPatch, name)
		}
		merged[name] = value
	}
	return merged, nil
}

func copyInputs(inputs map[string]any) map[string]any {
	copied := make(map[string]any, len(inputs))
	for name, value := range inputs {
		copied[name] = value
	}
	return copied
}

func missingInputs(required []string, inputs map[string]any) []string {
	var missing []string
	for _, name := range required {
		if !hasInput(inputs, name) {
			missing = append(missing, name)
		}
	}
	return missing
}

func requiresInput(required []string, target string) bool {
	for _, name := range required {
		if name == target {
			return true
		}
	}
	return false
}

func hasInput(inputs map[string]any, name string) bool {
	value, ok := inputs[name]
	return ok && value != nil && value != ""
}
