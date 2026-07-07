package use

import (
	"context"
	"errors"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
	runpkg "github.com/spacehz-lab/cal/internal/run"
	"github.com/spacehz-lab/cal/internal/use/plan"
	selector "github.com/spacehz-lab/cal/internal/use/select"
)

func TestRunRejectsMissingIntent(t *testing.T) {
	_, err := newUseRunner(newFakeStore()).Run(context.Background(), &Request{Inputs: map[string]any{}})
	if err == nil {
		t.Fatal("Run() error = nil, want invalid input error")
	}
}

func TestRunReturnsStoreError(t *testing.T) {
	store := newFakeStore()
	store.err = errors.New("list failed")
	_, err := newUseRunner(store).Run(context.Background(), &Request{Intent: "markdown to pdf"})
	if err == nil || err.Error() != "list failed" {
		t.Fatalf("Run() error = %v, want list failed", err)
	}
}

func TestRunReturnsNoMatchResult(t *testing.T) {
	var events []model.ProgressEvent
	result, err := newUseRunner(newFakeStore(), WithProgress(func(_ context.Context, event *model.ProgressEvent) {
		events = append(events, *event)
	})).Run(context.Background(), &Request{Intent: "resize image"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != model.RunStatusFailed || result.Error == nil || result.Error.Code != CodeNoMatch {
		t.Fatalf("result = %#v, want no_match failure", result)
	}
	assertUseProgress(t, events, []useProgressWant{
		{model.ProgressStageSelect, model.ProgressStatusStarted},
		{model.ProgressStageSelect, model.ProgressStatusFailed},
	})
}

func TestRunReturnsMissingInputsResult(t *testing.T) {
	store := newFakeStore()
	store.capabilities = []model.Capability{useCapability("markdown_to_pdf", "Markdown to PDF", useBinding("binding_pdf", []string{"run", "{{source}}"}))}
	result, err := newUseRunner(store).Run(context.Background(), &Request{Intent: "markdown pdf"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != model.RunStatusFailed || result.Error == nil || result.Error.Code != CodeMissingInputs {
		t.Fatalf("result = %#v, want missing_inputs failure", result)
	}
}

func TestRunDelegatesPlannedInputsToRun(t *testing.T) {
	store := newFakeStore()
	runRunner := &fakeRunRunner{result: &runpkg.Result{Run: &model.Run{ID: "run_1", Status: model.RunStatusSucceeded}}}
	var events []model.ProgressEvent
	runner := NewRunner(store, selector.NewRunner(), plan.NewRunner(), runRunner, WithProgress(func(_ context.Context, event *model.ProgressEvent) {
		events = append(events, *event)
	}))

	result, err := runner.Run(context.Background(), &Request{Intent: "markdown pdf", Inputs: map[string]any{"source": "input.md"}, Verify: true})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != model.RunStatusSucceeded || result.Run.ID != "run_1" {
		t.Fatalf("result = %#v, want succeeded run", result)
	}
	if runRunner.request == nil || runRunner.request.CapabilityID != "markdown_to_pdf" || runRunner.request.BindingID != "binding_pdf" {
		t.Fatalf("run request = %#v, want selected capability and binding", runRunner.request)
	}
	if runRunner.request.Inputs["source"] != "input.md" || runRunner.request.MinVerifyLevel != model.VerifyLevelL2 || !runRunner.request.Verify {
		t.Fatalf("run request = %#v, want planned inputs, default L2, verify", runRunner.request)
	}
	assertUseProgress(t, events, []useProgressWant{
		{model.ProgressStageSelect, model.ProgressStatusStarted},
		{model.ProgressStageSelect, model.ProgressStatusSucceeded},
		{model.ProgressStagePlan, model.ProgressStatusStarted},
		{model.ProgressStagePlan, model.ProgressStatusSucceeded},
		{model.ProgressStageRun, model.ProgressStatusStarted},
		{model.ProgressStageRun, model.ProgressStatusSucceeded},
	})
}

func TestRunMapsFailedRunResult(t *testing.T) {
	store := newFakeStore()
	runRunner := &fakeRunRunner{result: &runpkg.Result{Run: &model.Run{
		ID:     "run_failed",
		Status: model.RunStatusFailed,
		Error:  &model.RecordError{Code: runpkg.ErrorExecutionFailed, Message: "boom"},
	}}}
	result, err := NewRunner(store, selector.NewRunner(), plan.NewRunner(), runRunner).Run(context.Background(), &Request{
		Intent: "markdown pdf",
		Inputs: map[string]any{"source": "input.md"},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != model.RunStatusFailed || result.Error == nil || result.Error.Code != runpkg.ErrorExecutionFailed {
		t.Fatalf("result = %#v, want run failure code", result)
	}
}

func TestRunNormalizesNilInputs(t *testing.T) {
	store := newFakeStore()
	store.capabilities = []model.Capability{useCapability("markdown_to_pdf", "Markdown to PDF", useBinding("binding_pdf", []string{"run"}))}
	runRunner := &fakeRunRunner{result: &runpkg.Result{Run: &model.Run{ID: "run_1", Status: model.RunStatusSucceeded}}}
	_, err := NewRunner(store, selector.NewRunner(), plan.NewRunner(), runRunner).Run(context.Background(), &Request{Intent: "markdown pdf"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if runRunner.request.Inputs == nil {
		t.Fatal("run inputs = nil, want empty map")
	}
}

type fakeStore struct {
	capabilities []model.Capability
	err          error
}

func newFakeStore() *fakeStore {
	return &fakeStore{capabilities: []model.Capability{
		useCapability("markdown_to_pdf", "Markdown to PDF", useBinding("binding_pdf", []string{"run", "{{source}}"})),
	}}
}

func (store *fakeStore) ListCapabilities() ([]model.Capability, error) {
	if store.err != nil {
		return nil, store.err
	}
	return store.capabilities, nil
}

type fakeRunRunner struct {
	request *runpkg.Request
	result  *runpkg.Result
	err     error
}

func (runner *fakeRunRunner) Run(_ context.Context, req *runpkg.Request) (*runpkg.Result, error) {
	runner.request = req
	return runner.result, runner.err
}

func newUseRunner(store *fakeStore, opts ...Option) *Runner {
	return NewRunner(store, selector.NewRunner(), plan.NewRunner(), &fakeRunRunner{result: &runpkg.Result{Run: &model.Run{ID: "run_1", Status: model.RunStatusSucceeded}}}, opts...)
}

type useProgressWant struct {
	stage  model.ProgressStage
	status model.ProgressStatus
}

func assertUseProgress(t *testing.T, events []model.ProgressEvent, wants []useProgressWant) {
	t.Helper()
	if len(events) != len(wants) {
		t.Fatalf("progress len = %d, want %d: %#v", len(events), len(wants), events)
	}
	for index, want := range wants {
		event := events[index]
		if event.Scope != model.ProgressScopeUse || event.Stage != want.stage || event.Status != want.status {
			t.Fatalf("event[%d] = %#v, want %s/%s", index, event, want.stage, want.status)
		}
		if event.ID == "" || event.CreatedAt == "" || event.UseID == "" {
			t.Fatalf("event[%d] = %#v, want id, created_at, and use_id", index, event)
		}
	}
}

func useCapability(id string, description string, bindings ...model.Binding) model.Capability {
	for index := range bindings {
		bindings[index].CapabilityID = id
	}
	return model.Capability{ID: id, Description: description, Bindings: bindings}
}

func useBinding(id string, args []string) model.Binding {
	return model.Binding{
		ID:         id,
		ProviderID: "provider_test",
		Execution:  model.Execution{Kind: model.ExecutionKindCLI, Spec: map[string]any{model.ExecutionSpecArgs: args}},
		Verify: &model.VerifySpec{
			Level:  model.VerifyLevelL2,
			Method: model.VerifyMethodExecute,
			Checks: []model.VerifyCheck{{
				Subject:   model.VerifySubject{Type: model.VerifySubjectStdout},
				Predicate: model.VerifyPredicateNonEmpty,
			}},
		},
		Evidence: []model.EvidenceRef{{ID: "evidence_1"}},
		State:    model.BindingStatePromoted,
	}
}
