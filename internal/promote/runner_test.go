package promote

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestRunnerRunCreatesCapability(t *testing.T) {
	store := newFakeStore()
	runner := NewRunner(store, fixedNow)

	result, err := runner.Run(context.Background(), request(candidateWithArgs("run"), passedProbe(0)))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(result.Promotions) != 1 {
		t.Fatalf("promotions len = %d, want 1", len(result.Promotions))
	}
	promotion := result.Promotions[0]
	if promotion.CapabilityAction != ActionCreated || promotion.BindingAction != ActionCreated {
		t.Fatalf("promotion = %#v, want created/created", promotion)
	}
	capability := store.capabilities["document.convert"]
	if capability.ID != "document.convert" || len(capability.Bindings) != 1 {
		t.Fatalf("saved capability = %#v, want one binding", capability)
	}
	if capability.Bindings[0].State != model.BindingStatePromoted || capability.Bindings[0].Verify == nil || len(capability.Bindings[0].Evidence) != 1 {
		t.Fatalf("saved binding = %#v, want promoted verify evidence", capability.Bindings[0])
	}
}

func TestRunnerRunAppendsDifferentBinding(t *testing.T) {
	store := newFakeStore()
	existing := capabilityWithBinding(t, candidateWithArgs("old"), []model.EvidenceRef{{ID: "old_evidence"}})
	store.capabilities[existing.ID] = existing
	runner := NewRunner(store, fixedNow)

	result, err := runner.Run(context.Background(), request(candidateWithArgs("new"), passedProbe(0)))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Promotions[0].CapabilityAction != ActionReused || result.Promotions[0].BindingAction != ActionCreated {
		t.Fatalf("promotion = %#v, want reused/created", result.Promotions[0])
	}
	saved := store.capabilities["document.convert"]
	if len(saved.Bindings) != 2 {
		t.Fatalf("bindings len = %d, want 2", len(saved.Bindings))
	}
	if saved.Description != existing.Description {
		t.Fatalf("description = %q, want existing description", saved.Description)
	}
}

func TestRunnerRunUpdatesSameBinding(t *testing.T) {
	store := newFakeStore()
	candidate := candidateWithArgs("same")
	existing := capabilityWithBinding(t, candidate, []model.EvidenceRef{{ID: "old_evidence"}})
	store.capabilities[existing.ID] = existing
	runner := NewRunner(store, fixedNow)

	result, err := runner.Run(context.Background(), request(candidate, passedProbe(0)))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Promotions[0].CapabilityAction != ActionReused || result.Promotions[0].BindingAction != ActionUpdated {
		t.Fatalf("promotion = %#v, want reused/updated", result.Promotions[0])
	}
	saved := store.capabilities["document.convert"]
	if len(saved.Bindings) != 1 || saved.Bindings[0].Evidence[0].ID != "evidence_file_exists" {
		t.Fatalf("saved bindings = %#v, want replacement binding", saved.Bindings)
	}
}

func TestRunnerRunSkipsFailedProbe(t *testing.T) {
	store := newFakeStore()
	runner := NewRunner(store, fixedNow)

	result, err := runner.Run(context.Background(), request(candidateWithArgs("run"), model.Probe{CandidateIndex: 0, Passed: false}))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(result.Promotions) != 0 {
		t.Fatalf("promotions len = %d, want 0", len(result.Promotions))
	}
	if len(store.capabilities) != 0 {
		t.Fatalf("saved capabilities = %#v, want none", store.capabilities)
	}
}

func TestRunnerRunRejectsInvalidPassedProbe(t *testing.T) {
	store := newFakeStore()
	runner := NewRunner(store, fixedNow)
	bad := candidateWithArgs("run")
	bad.Description = ""

	_, err := runner.Run(context.Background(), request(bad, passedProbe(0)))
	if err == nil {
		t.Fatal("Run() error = nil, want rejection")
	}
	var promoteErr *Error
	if !errors.As(err, &promoteErr) || promoteErr.Code != CodePromotionRejected {
		t.Fatalf("Run() error = %#v, want CodePromotionRejected", err)
	}
}

func TestRunnerRunRejectsL0PassedProbe(t *testing.T) {
	store := newFakeStore()
	runner := NewRunner(store, fixedNow)
	probe := passedProbe(0)
	probe.Verify.Level = model.VerifyLevelL0

	_, err := runner.Run(context.Background(), request(candidateWithArgs("run"), probe))
	if err == nil {
		t.Fatal("Run() error = nil, want L0 rejection")
	}
	var promoteErr *Error
	if !errors.As(err, &promoteErr) || promoteErr.Code != CodePromotionRejected {
		t.Fatalf("Run() error = %#v, want CodePromotionRejected", err)
	}
}

func TestRunnerRunRejectsEmptyVerifyShape(t *testing.T) {
	store := newFakeStore()
	runner := NewRunner(store, fixedNow)
	probe := passedProbe(0)
	probe.Verify = model.VerifySpec{}

	_, err := runner.Run(context.Background(), request(candidateWithArgs("run"), probe))
	if err == nil {
		t.Fatal("Run() error = nil, want verify shape rejection")
	}
	var promoteErr *Error
	if !errors.As(err, &promoteErr) || promoteErr.Code != CodePromotionRejected {
		t.Fatalf("Run() error = %#v, want CodePromotionRejected", err)
	}
}

func TestRunnerRunReturnsStoreLoadFailure(t *testing.T) {
	store := newFakeStore()
	store.getErr = errors.New("load failed")
	runner := NewRunner(store, fixedNow)

	_, err := runner.Run(context.Background(), request(candidateWithArgs("run"), passedProbe(0)))
	if err == nil {
		t.Fatal("Run() error = nil, want store error")
	}
	var promoteErr *Error
	if !errors.As(err, &promoteErr) || promoteErr.Code != CodePromotionStoreFailed {
		t.Fatalf("Run() error = %#v, want CodePromotionStoreFailed", err)
	}
}

func TestRunnerRunReturnsStoreSaveFailure(t *testing.T) {
	store := newFakeStore()
	store.saveErr = errors.New("save failed")
	runner := NewRunner(store, fixedNow)

	_, err := runner.Run(context.Background(), request(candidateWithArgs("run"), passedProbe(0)))
	if err == nil {
		t.Fatal("Run() error = nil, want store error")
	}
	var promoteErr *Error
	if !errors.As(err, &promoteErr) || promoteErr.Code != CodePromotionStoreFailed {
		t.Fatalf("Run() error = %#v, want CodePromotionStoreFailed", err)
	}
}

func TestRunnerRunRejectsDuplicatePassedProbe(t *testing.T) {
	store := newFakeStore()
	runner := NewRunner(store, fixedNow)
	req := request(candidateWithArgs("run"), passedProbe(0))
	req.Probes = append(req.Probes, passedProbe(0))

	_, err := runner.Run(context.Background(), req)
	if err == nil {
		t.Fatal("Run() error = nil, want duplicate index error")
	}
	var promoteErr *Error
	if !errors.As(err, &promoteErr) || promoteErr.Code != CodeInvalidPromotionInput {
		t.Fatalf("Run() error = %#v, want CodeInvalidPromotionInput", err)
	}
}

func request(candidate model.Candidate, probe model.Probe) *Request {
	return &Request{Candidates: []model.Candidate{candidate}, Probes: []model.Probe{probe}}
}

func candidateWithArgs(arg string) model.Candidate {
	return model.Candidate{
		ProviderID:   "provider_cli",
		CapabilityID: "document.convert",
		Description:  "Convert a document.",
		Execution: model.Execution{
			Kind: model.ExecutionKindCLI,
			Spec: map[string]any{model.ExecutionSpecArgs: []string{arg}},
		},
	}
}

func passedProbe(index int) model.Probe {
	return model.Probe{
		CandidateIndex: index,
		Passed:         true,
		Verify:         fileExistsVerify(),
		Evidence:       []model.EvidenceRef{{ID: "evidence_file_exists"}},
	}
}

func fileExistsVerify() model.VerifySpec {
	return model.VerifySpec{
		Level:  model.VerifyLevelL1,
		Method: model.VerifyMethodExecute,
		Checks: []model.VerifyCheck{{
			Subject:   model.VerifySubject{Type: model.VerifySubjectFile, Input: "target"},
			Predicate: model.VerifyPredicateExists,
		}},
	}
}

func capabilityWithBinding(t *testing.T, candidate model.Candidate, evidence []model.EvidenceRef) model.Capability {
	t.Helper()
	bindingID, err := model.BindingIDForExecution(candidate.CapabilityID, candidate.ProviderID, candidate.Execution)
	if err != nil {
		t.Fatalf("BindingIDForExecution() error = %v", err)
	}
	verify := fileExistsVerify()
	return model.Capability{
		ID:          candidate.CapabilityID,
		Description: "Existing description.",
		Bindings: []model.Binding{{
			ID:           bindingID,
			CapabilityID: candidate.CapabilityID,
			ProviderID:   candidate.ProviderID,
			Execution:    candidate.Execution,
			Verify:       &verify,
			Evidence:     evidence,
			State:        model.BindingStatePromoted,
		}},
	}
}

func fixedNow() time.Time {
	return time.Unix(100, 0).UTC()
}

type fakeStore struct {
	capabilities map[string]model.Capability
	getErr       error
	saveErr      error
}

func newFakeStore() *fakeStore {
	return &fakeStore{capabilities: map[string]model.Capability{}}
}

func (store *fakeStore) GetCapability(id string) (model.Capability, bool, error) {
	if store.getErr != nil {
		return model.Capability{}, false, store.getErr
	}
	capability, ok := store.capabilities[id]
	return capability, ok, nil
}

func (store *fakeStore) SaveCapability(capability *model.Capability) error {
	if store.saveErr != nil {
		return store.saveErr
	}
	store.capabilities[capability.ID] = *capability
	return nil
}
