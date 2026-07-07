package resolve

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spacehz-lab/cal/internal/execute"
	"github.com/spacehz-lab/cal/internal/model"
)

var (
	ErrNilRequest      = errors.New("resolve request is required")
	ErrNilCapability   = errors.New("resolve capability is required")
	ErrBindingNotFound = errors.New("binding not found")
)

// Request provides deterministic binding selection input.
type Request struct {
	Capability     *model.Capability
	BindingID      string
	ProviderID     string
	Inputs         map[string]any
	MinVerifyLevel model.VerifyLevel
}

// Result describes the selected binding and its execution inputs.
type Result struct {
	Capability     *model.Capability
	Binding        *model.Binding
	RequiredInputs []string
}

// Runner selects promoted bindings for known capabilities.
type Runner struct{}

// NewRunner creates a binding resolver.
func NewRunner() *Runner {
	return &Runner{}
}

// Run selects one promoted binding.
func (runner *Runner) Run(req *Request) (*Result, error) {
	if req == nil {
		return nil, ErrNilRequest
	}
	if req.Capability == nil {
		return nil, ErrNilCapability
	}
	var first *Result
	for index := range req.Capability.Bindings {
		binding := &req.Capability.Bindings[index]
		if !eligible(binding, req) {
			continue
		}
		required, err := execute.RequiredInputs(&binding.Execution)
		if err != nil {
			return nil, err
		}
		result := &Result{
			Capability:     req.Capability,
			Binding:        binding,
			RequiredInputs: required,
		}
		if first == nil {
			first = result
		}
		if inputsSatisfied(required, req.Inputs) {
			return result, nil
		}
	}
	if first != nil {
		return first, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrBindingNotFound, req.Capability.ID)
}

func eligible(binding *model.Binding, req *Request) bool {
	if binding.State != model.BindingStatePromoted {
		return false
	}
	if strings.TrimSpace(req.BindingID) != "" && binding.ID != strings.TrimSpace(req.BindingID) {
		return false
	}
	if strings.TrimSpace(req.ProviderID) != "" && binding.ProviderID != strings.TrimSpace(req.ProviderID) {
		return false
	}
	if req.MinVerifyLevel != "" {
		if binding.Verify == nil {
			return false
		}
		if model.VerifyLevelRank(binding.Verify.Level) < model.VerifyLevelRank(req.MinVerifyLevel) {
			return false
		}
	}
	return true
}

func inputsSatisfied(required []string, inputs map[string]any) bool {
	for _, name := range required {
		value, ok := inputs[name]
		if !ok || value == nil || value == "" {
			return false
		}
	}
	return true
}
