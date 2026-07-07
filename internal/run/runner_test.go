package run

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/check"
	"github.com/spacehz-lab/cal/internal/execute"
	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/run/resolve"
)

func TestRunSucceedsWithoutVerification(t *testing.T) {
	store := newFakeStore()
	executor := &fakeExecutor{result: executeResult(0, "ok\n", "")}
	runner := NewRunner(store, resolve.NewRunner(), executor, nil)

	result, err := runner.Run(context.Background(), &Request{CapabilityID: "capability_test", Inputs: map[string]any{}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Run.Status != model.RunStatusSucceeded {
		t.Fatalf("status = %s, want succeeded", result.Run.Status)
	}
	if result.Run.Verified {
		t.Fatal("verified = true, want false")
	}
	if len(store.saved) != 1 {
		t.Fatalf("saved runs = %d, want 1", len(store.saved))
	}
}

func TestRunFailsWhenCapabilityMissing(t *testing.T) {
	store := newFakeStore()
	delete(store.capabilities, "capability_test")
	runner := NewRunner(store, resolve.NewRunner(), &fakeExecutor{result: executeResult(0, "", "")}, nil)

	result, err := runner.Run(context.Background(), &Request{CapabilityID: "capability_test", Inputs: map[string]any{}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	assertFailedRun(t, result.Run, ErrorCapabilityNotFound)
}

func TestRunFailsWhenProviderMissing(t *testing.T) {
	store := newFakeStore()
	delete(store.providers, "provider_test")
	runner := NewRunner(store, resolve.NewRunner(), &fakeExecutor{result: executeResult(0, "", "")}, nil)

	result, err := runner.Run(context.Background(), &Request{CapabilityID: "capability_test", Inputs: map[string]any{}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	assertFailedRun(t, result.Run, ErrorProviderNotFound)
}

func TestRunFailsWhenRequiredInputMissing(t *testing.T) {
	store := newFakeStore()
	store.capabilities["capability_test"] = capability([]string{"run", "{{source}}"}, true)
	runner := NewRunner(store, resolve.NewRunner(), &fakeExecutor{result: executeResult(0, "", "")}, nil)

	result, err := runner.Run(context.Background(), &Request{CapabilityID: "capability_test", Inputs: map[string]any{}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	assertFailedRun(t, result.Run, ErrorInvalidRunInput)
}

func TestRunFailsWhenExecuteReturnsError(t *testing.T) {
	store := newFakeStore()
	runner := NewRunner(store, resolve.NewRunner(), &fakeExecutor{err: errors.New("boom")}, nil)

	result, err := runner.Run(context.Background(), &Request{CapabilityID: "capability_test", Inputs: map[string]any{}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	assertFailedRun(t, result.Run, ErrorExecutionFailed)
}

func TestRunFailsWhenExecuteReturnsNilResult(t *testing.T) {
	store := newFakeStore()
	runner := NewRunner(store, resolve.NewRunner(), &fakeExecutor{}, nil)

	result, err := runner.Run(context.Background(), &Request{CapabilityID: "capability_test", Inputs: map[string]any{}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	assertFailedRun(t, result.Run, ErrorExecutionFailed)
}

func TestRunFailsOnNonZeroExitWithoutVerification(t *testing.T) {
	store := newFakeStore()
	runner := NewRunner(store, resolve.NewRunner(), &fakeExecutor{result: executeResult(7, "", "failed\n")}, nil)

	result, err := runner.Run(context.Background(), &Request{CapabilityID: "capability_test", Inputs: map[string]any{}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	assertFailedRun(t, result.Run, ErrorExecutionFailed)
}

func TestRunSucceedsWithVerification(t *testing.T) {
	store := newFakeStore()
	executor := &fakeExecutor{result: executeResult(0, "ok\n", "")}
	checker := &fakeChecker{result: &check.Result{
		Evidence: []model.EvidenceRef{{ID: "check_1"}},
		Outputs:  map[string]any{"stdout": "ok\n"},
	}}
	runner := NewRunner(store, resolve.NewRunner(), executor, checker)

	result, err := runner.Run(context.Background(), &Request{CapabilityID: "capability_test", Inputs: map[string]any{}, Verify: true})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Run.Status != model.RunStatusSucceeded || !result.Run.Verified {
		t.Fatalf("run = %#v, want verified success", result.Run)
	}
	if len(result.Evidence) != 1 || result.Evidence[0].ID != "check_1" {
		t.Fatalf("evidence = %#v, want check_1", result.Evidence)
	}
	if checker.request.Stdout != "ok\n" || checker.request.ExitCode != 0 {
		t.Fatalf("check request = %#v, want stdout and exit code", checker.request)
	}
}

func TestRunEmitsProgress(t *testing.T) {
	store := newFakeStore()
	executor := &fakeExecutor{result: executeResult(0, "ok\n", "")}
	checker := &fakeChecker{result: &check.Result{Evidence: []model.EvidenceRef{{ID: "check_1"}}}}
	var events []model.ProgressEvent
	runner := NewRunner(store, resolve.NewRunner(), executor, checker, WithProgress(func(_ context.Context, event *model.ProgressEvent) {
		events = append(events, *event)
	}))

	if _, err := runner.Run(context.Background(), &Request{CapabilityID: "capability_test", Inputs: map[string]any{}, Verify: true}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	assertRunProgress(t, events, []runProgressWant{
		{model.ProgressStageResolve, model.ProgressStatusStarted},
		{model.ProgressStageResolve, model.ProgressStatusSucceeded},
		{model.ProgressStageExecute, model.ProgressStatusStarted},
		{model.ProgressStageExecute, model.ProgressStatusSucceeded},
		{model.ProgressStageVerify, model.ProgressStatusStarted},
		{model.ProgressStageVerify, model.ProgressStatusSucceeded},
		{model.ProgressStageRecord, model.ProgressStatusStarted},
		{model.ProgressStageRecord, model.ProgressStatusSucceeded},
	})
}

func TestRunFailsWhenVerifyRequestedWithoutSpec(t *testing.T) {
	store := newFakeStore()
	store.capabilities["capability_test"] = capability([]string{"run"}, false)
	runner := NewRunner(store, resolve.NewRunner(), &fakeExecutor{result: executeResult(0, "ok\n", "")}, &fakeChecker{})

	result, err := runner.Run(context.Background(), &Request{CapabilityID: "capability_test", Inputs: map[string]any{}, Verify: true})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	assertFailedRun(t, result.Run, ErrorVerificationFailed)
}

func TestRunFailsWhenCheckReturnsError(t *testing.T) {
	store := newFakeStore()
	runner := NewRunner(
		store,
		resolve.NewRunner(),
		&fakeExecutor{result: executeResult(0, "bad\n", "")},
		&fakeChecker{err: errors.New("check failed")},
	)

	result, err := runner.Run(context.Background(), &Request{CapabilityID: "capability_test", Inputs: map[string]any{}, Verify: true})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	assertFailedRun(t, result.Run, ErrorVerificationFailed)
}

func TestRunReturnsSaveRunError(t *testing.T) {
	store := newFakeStore()
	store.saveErr = errors.New("disk full")
	runner := NewRunner(store, resolve.NewRunner(), &fakeExecutor{result: executeResult(0, "", "")}, nil)

	_, err := runner.Run(context.Background(), &Request{CapabilityID: "capability_test", Inputs: map[string]any{}})
	if err == nil || !strings.Contains(err.Error(), ErrorRunStoreFailed) {
		t.Fatalf("Run() error = %v, want store failure", err)
	}
}

func assertFailedRun(t *testing.T, run *model.Run, code string) {
	t.Helper()
	if run.Status != model.RunStatusFailed {
		t.Fatalf("status = %s, want failed", run.Status)
	}
	if run.Error == nil || run.Error.Code != code {
		t.Fatalf("error = %#v, want code %s", run.Error, code)
	}
}

type runProgressWant struct {
	stage  model.ProgressStage
	status model.ProgressStatus
}

func assertRunProgress(t *testing.T, events []model.ProgressEvent, wants []runProgressWant) {
	t.Helper()
	if len(events) != len(wants) {
		t.Fatalf("progress len = %d, want %d: %#v", len(events), len(wants), events)
	}
	for index, want := range wants {
		event := events[index]
		if event.Scope != model.ProgressScopeRun || event.Stage != want.stage || event.Status != want.status {
			t.Fatalf("event[%d] = %#v, want %s/%s", index, event, want.stage, want.status)
		}
		if event.ID == "" || event.CreatedAt == "" || event.RunID == "" {
			t.Fatalf("event[%d] = %#v, want id, created_at, and run_id", index, event)
		}
	}
}

type fakeStore struct {
	capabilities map[string]model.Capability
	providers    map[string]model.Provider
	saved        []model.Run
	saveErr      error
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		capabilities: map[string]model.Capability{"capability_test": capability([]string{"run"}, true)},
		providers: map[string]model.Provider{"provider_test": {
			ID:   "provider_test",
			Kind: model.ProviderKindCLI,
			Path: "/bin/echo",
		}},
	}
}

func (store *fakeStore) GetCapability(id string) (model.Capability, bool, error) {
	capability, ok := store.capabilities[id]
	return capability, ok, nil
}

func (store *fakeStore) GetProvider(id string) (model.Provider, bool, error) {
	provider, ok := store.providers[id]
	return provider, ok, nil
}

func (store *fakeStore) SaveRun(run *model.Run) error {
	if store.saveErr != nil {
		return store.saveErr
	}
	store.saved = append(store.saved, *run)
	return nil
}

type fakeExecutor struct {
	result *execute.Result
	err    error
}

func (executor *fakeExecutor) Run(context.Context, *execute.Request) (*execute.Result, error) {
	return executor.result, executor.err
}

type fakeChecker struct {
	request *check.Request
	result  *check.Result
	err     error
}

func (checker *fakeChecker) Run(_ context.Context, req *check.Request) (*check.Result, error) {
	checker.request = req
	return checker.result, checker.err
}

func capability(args []string, withVerify bool) model.Capability {
	binding := model.Binding{
		ID:           "binding_test",
		CapabilityID: "capability_test",
		ProviderID:   "provider_test",
		Execution:    model.Execution{Kind: model.ExecutionKindCLI, Spec: map[string]any{model.ExecutionSpecArgs: args}},
		Evidence:     []model.EvidenceRef{{ID: "evidence_1"}},
		State:        model.BindingStatePromoted,
	}
	if withVerify {
		binding.Verify = &model.VerifySpec{
			Level:  model.VerifyLevelL1,
			Method: model.VerifyMethodExecute,
			Checks: []model.VerifyCheck{{
				Subject:   model.VerifySubject{Type: model.VerifySubjectStdout},
				Predicate: model.VerifyPredicateNonEmpty,
			}},
		}
	}
	return model.Capability{ID: "capability_test", Description: "test capability", Bindings: []model.Binding{binding}}
}

func executeResult(exitCode int, stdout string, stderr string) *execute.Result {
	return &execute.Result{Outputs: execute.Outputs{
		execute.OutputStdout:   {Kind: execute.OutputKindText, Text: stdout},
		execute.OutputStderr:   {Kind: execute.OutputKindText, Text: stderr},
		execute.OutputExitCode: {Kind: execute.OutputKindNumber, Number: &exitCode},
	}}
}
