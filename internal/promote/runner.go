package promote

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spacehz-lab/cal/internal/model"
)

// Runner promotes passed candidate probes into durable capability bindings.
type Runner struct {
	store Store
	now   func() time.Time
}

// NewRunner creates a promotion runner.
func NewRunner(store Store, now func() time.Time) *Runner {
	return &Runner{store: store, now: now}
}

// Run promotes all passed probes in the request.
func (runner *Runner) Run(ctx context.Context, req *Request) (*Result, error) {
	if err := runner.validate(req); err != nil {
		return nil, err
	}
	targets, err := buildTargets(req)
	if err != nil {
		return nil, err
	}
	result := &Result{Promotions: make([]model.Promotion, 0, len(targets))}
	for index := range targets {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		promotion, err := runner.promoteTarget(&targets[index], nowUTC(runner.now))
		if err != nil {
			return result, err
		}
		result.Promotions = append(result.Promotions, promotion)
	}
	return result, nil
}

func (runner *Runner) validate(req *Request) error {
	if runner == nil {
		return newError(CodeInvalidPromotionInput, "promotion runner is required")
	}
	if runner.store == nil {
		return newError(CodeInvalidPromotionInput, "promotion store is required")
	}
	if req == nil {
		return newError(CodeInvalidPromotionInput, "promotion request is required")
	}
	return nil
}

func (runner *Runner) promoteTarget(target *Target, now time.Time) (model.Promotion, error) {
	promoted, binding, err := promotedCapability(target.Candidate, target.Probe, now)
	if err != nil {
		return model.Promotion{}, err
	}
	existing, exists, err := runner.store.GetCapability(promoted.ID)
	if err != nil {
		return model.Promotion{}, wrapError(CodePromotionStoreFailed, "load capability", err)
	}
	capabilityAction := ActionCreated
	bindingAction := ActionCreated
	capability := promoted
	if exists {
		capabilityAction = ActionReused
		if hasBinding(existing, binding.ID) {
			bindingAction = ActionUpdated
		}
		capability = mergeCapability(existing, promoted)
	}
	if err := runner.store.SaveCapability(&capability); err != nil {
		return model.Promotion{}, wrapError(CodePromotionStoreFailed, "save capability", err)
	}
	return model.Promotion{
		CandidateIndex:   target.CandidateIndex,
		CapabilityID:     capability.ID,
		BindingID:        binding.ID,
		ProviderID:       binding.ProviderID,
		CapabilityAction: capabilityAction,
		BindingAction:    bindingAction,
		CreatedAt:        now.Format(time.RFC3339Nano),
	}, nil
}

func buildTargets(req *Request) ([]Target, error) {
	seen := map[int]struct{}{}
	targets := make([]Target, 0, len(req.Probes))
	for index := range req.Probes {
		probe := &req.Probes[index]
		if !probe.Passed {
			continue
		}
		if probe.CandidateIndex < 0 || probe.CandidateIndex >= len(req.Candidates) {
			return nil, newError(CodeInvalidPromotionInput, "passed probe candidate index is out of range")
		}
		if _, ok := seen[probe.CandidateIndex]; ok {
			return nil, newError(CodeInvalidPromotionInput, "duplicate passed probe candidate index")
		}
		seen[probe.CandidateIndex] = struct{}{}
		targets = append(targets, Target{
			CandidateIndex: probe.CandidateIndex,
			Candidate:      &req.Candidates[probe.CandidateIndex],
			Probe:          probe,
		})
	}
	return targets, nil
}

func promotedCapability(candidate *model.Candidate, probe *model.Probe, now time.Time) (model.Capability, model.Binding, error) {
	if candidate == nil || probe == nil {
		return model.Capability{}, model.Binding{}, newError(CodePromotionRejected, "candidate and probe are required")
	}
	if !probe.Passed {
		return model.Capability{}, model.Binding{}, newError(CodePromotionRejected, "cannot promote failed probe")
	}
	providerID := strings.TrimSpace(candidate.ProviderID)
	capabilityID := strings.TrimSpace(candidate.CapabilityID)
	description := strings.TrimSpace(candidate.Description)
	switch {
	case providerID == "":
		return model.Capability{}, model.Binding{}, newError(CodePromotionRejected, "candidate provider id is required")
	case capabilityID == "":
		return model.Capability{}, model.Binding{}, newError(CodePromotionRejected, "candidate capability id is required")
	case !model.ValidCapabilityID(capabilityID):
		return model.Capability{}, model.Binding{}, newError(CodePromotionRejected, fmt.Sprintf("candidate capability id %q is invalid", capabilityID))
	case description == "":
		return model.Capability{}, model.Binding{}, newError(CodePromotionRejected, "candidate description is required")
	case candidate.Execution.Kind == "":
		return model.Capability{}, model.Binding{}, newError(CodePromotionRejected, "candidate execution is required")
	case probe.Verify.Level == "":
		return model.Capability{}, model.Binding{}, newError(CodePromotionRejected, "probe verify level is required")
	case probe.Verify.Level == model.VerifyLevelL0:
		return model.Capability{}, model.Binding{}, newError(CodePromotionRejected, "cannot promote L0 verification")
	case probe.Verify.Method == "":
		return model.Capability{}, model.Binding{}, newError(CodePromotionRejected, "probe verify method is required")
	case len(probe.Evidence) == 0:
		return model.Capability{}, model.Binding{}, newError(CodePromotionRejected, "probe evidence is required")
	}
	bindingID, err := model.BindingIDForExecution(capabilityID, providerID, candidate.Execution)
	if err != nil {
		return model.Capability{}, model.Binding{}, wrapError(CodePromotionRejected, "derive binding id", err)
	}
	verify := probe.Verify
	binding := model.Binding{
		ID:           bindingID,
		CapabilityID: capabilityID,
		ProviderID:   providerID,
		Execution:    candidate.Execution,
		Verify:       &verify,
		Evidence:     append([]model.EvidenceRef(nil), probe.Evidence...),
		State:        model.BindingStatePromoted,
		CreatedAt:    now.Format(time.RFC3339Nano),
	}
	capability := model.Capability{
		ID:          capabilityID,
		Description: description,
		Bindings:    []model.Binding{binding},
	}
	if err := model.ValidateCapability(capability); err != nil {
		return model.Capability{}, model.Binding{}, wrapError(CodePromotionRejected, "validate promoted capability", err)
	}
	return capability, binding, nil
}

func mergeCapability(existing model.Capability, promoted model.Capability) model.Capability {
	if strings.TrimSpace(existing.Description) == "" {
		existing.Description = promoted.Description
	}
	bindings := make([]model.Binding, 0, len(existing.Bindings)+len(promoted.Bindings))
	replaced := map[string]struct{}{}
	for _, existingBinding := range existing.Bindings {
		replacement, ok := bindingByID(promoted.Bindings, existingBinding.ID)
		if ok {
			bindings = append(bindings, replacement)
			replaced[existingBinding.ID] = struct{}{}
			continue
		}
		bindings = append(bindings, existingBinding)
	}
	for _, promotedBinding := range promoted.Bindings {
		if _, ok := replaced[promotedBinding.ID]; ok {
			continue
		}
		bindings = append(bindings, promotedBinding)
	}
	existing.Bindings = bindings
	return existing
}

func hasBinding(capability model.Capability, bindingID string) bool {
	_, ok := bindingByID(capability.Bindings, bindingID)
	return ok
}

func bindingByID(bindings []model.Binding, id string) (model.Binding, bool) {
	for _, binding := range bindings {
		if binding.ID == id {
			return binding, true
		}
	}
	return model.Binding{}, false
}
