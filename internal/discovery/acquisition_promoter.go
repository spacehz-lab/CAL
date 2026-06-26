package discovery

import (
	"fmt"
	"strings"
	"time"

	"github.com/spacehz-lab/cal/internal/core"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

type acquisitionPromoter struct {
	store    Store
	provider core.Provider
	now      time.Time
}

func newAcquisitionPromoter(store Store, provider core.Provider, now time.Time) acquisitionPromoter {
	return acquisitionPromoter{
		store:    store,
		provider: provider,
		now:      now,
	}
}

func (promoter acquisitionPromoter) Promote(candidate caltrace.Candidate, probe caltrace.Probe, candidateIndex int) (caltrace.Promotion, error) {
	capability, binding, err := promoter.promotedCapability(candidate, probe)
	if err != nil {
		return caltrace.Promotion{}, err
	}

	existing, exists, err := promoter.store.GetCapability(capability.ID)
	if err != nil {
		return caltrace.Promotion{}, err
	}
	capabilityAction := "created"
	bindingAction := "created"
	if exists {
		capabilityAction = "reused"
		if promoter.hasBinding(existing, binding.ID) {
			bindingAction = "updated"
		}
		capability = promoter.mergeCapability(existing, capability)
	}
	if err := promoter.store.PutCapability(capability); err != nil {
		return caltrace.Promotion{}, err
	}
	promotion := caltrace.Promotion{
		CandidateIndex:   candidateIndex,
		CapabilityID:     capability.ID,
		BindingID:        binding.ID,
		ProviderID:       promoter.provider.ID,
		CapabilityAction: capabilityAction,
		BindingAction:    bindingAction,
		CreatedAt:        promoter.now.UTC().Format(time.RFC3339Nano),
	}
	return promotion, nil
}

func (promoter acquisitionPromoter) promotedCapability(candidate caltrace.Candidate, probe caltrace.Probe) (core.Capability, core.Binding, error) {
	if !probe.Passed {
		return core.Capability{}, core.Binding{}, fmt.Errorf("cannot promote failed or ambiguous probe")
	}
	if candidate.ProviderID == "" || candidate.CapabilityID == "" {
		return core.Capability{}, core.Binding{}, fmt.Errorf("candidate provider_id and capability_id are required")
	}
	description := strings.TrimSpace(candidate.Description)
	if description == "" {
		return core.Capability{}, core.Binding{}, fmt.Errorf("candidate description is required")
	}
	if !core.ValidCapabilityID(candidate.CapabilityID) {
		return core.Capability{}, core.Binding{}, fmt.Errorf("candidate capability_id %q is invalid", candidate.CapabilityID)
	}
	if candidate.Execution.Kind == "" {
		return core.Capability{}, core.Binding{}, fmt.Errorf("candidate execution is required")
	}
	if probe.Verifier.ID == "" {
		return core.Capability{}, core.Binding{}, fmt.Errorf("probe verifier is required")
	}
	if len(probe.Evidence) == 0 {
		return core.Capability{}, core.Binding{}, fmt.Errorf("probe evidence is required")
	}
	bindingID, err := core.BindingIDForExecution(candidate.CapabilityID, candidate.ProviderID, candidate.Execution)
	if err != nil {
		return core.Capability{}, core.Binding{}, err
	}

	binding := core.Binding{
		ID:               bindingID,
		CapabilityID:     candidate.CapabilityID,
		ProviderID:       candidate.ProviderID,
		InputConstraints: candidate.InputConstraints,
		Execution:        candidate.Execution,
		Verifier:         &probe.Verifier,
		Evidence:         probe.Evidence,
		State:            core.BindingStatePromoted,
	}
	capability := core.Capability{
		ID:          candidate.CapabilityID,
		Description: description,
		Bindings:    []core.Binding{binding},
	}
	if err := core.ValidateCapability(capability); err != nil {
		return core.Capability{}, core.Binding{}, err
	}
	return capability, binding, nil
}

func (promoter acquisitionPromoter) mergeCapability(existing, promoted core.Capability) core.Capability {
	if existing.ID == "" {
		return promoted
	}
	if strings.TrimSpace(existing.Description) == "" {
		existing.Description = promoted.Description
	}
	bindings := make([]core.Binding, 0, len(existing.Bindings)+len(promoted.Bindings))
	replaced := map[string]struct{}{}
	for _, existingBinding := range existing.Bindings {
		replacement, ok := promoter.bindingByID(promoted.Bindings, existingBinding.ID)
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

func (promoter acquisitionPromoter) hasBinding(capability core.Capability, bindingID string) bool {
	for _, binding := range capability.Bindings {
		if binding.ID == bindingID {
			return true
		}
	}
	return false
}

func (promoter acquisitionPromoter) bindingByID(bindings []core.Binding, id string) (core.Binding, bool) {
	for _, binding := range bindings {
		if binding.ID == id {
			return binding, true
		}
	}
	return core.Binding{}, false
}
