package observe

import (
	"context"
	"errors"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestObserveRejectsMissingProvider(t *testing.T) {
	_, err := NewRunner(nil).Observe(context.Background(), &Request{})
	assertObserveCode(t, err, CodeInvalidObserveInput)

	_, err = NewRunner(nil).Observe(context.Background(), nil)
	assertObserveCode(t, err, CodeInvalidObserveInput)
}

func TestObserveRejectsUnsupportedAndUnconfiguredKind(t *testing.T) {
	_, err := NewRunner(nil).Observe(context.Background(), &Request{Provider: &model.Provider{ID: "provider_test"}})
	assertObserveCode(t, err, CodeUnsupportedProviderKind)

	_, err = NewRunner(nil).Observe(context.Background(), &Request{Provider: &model.Provider{ID: "provider_test", Kind: model.ProviderKindApp}})
	assertObserveCode(t, err, CodeObserverNotConfigured)
}

func TestObserveDispatchesAndCopiesObserverMap(t *testing.T) {
	first := &fakeObserver{result: &Result{Observations: []model.Observation{{Type: ObservationTypeCLIOutput}}}}
	second := &fakeObserver{result: &Result{}}
	observers := map[model.ProviderKind]Observer{model.ProviderKindCLI: first}
	runner := NewRunner(observers)
	observers[model.ProviderKindCLI] = second

	result, err := runner.Observe(context.Background(), &Request{Provider: cliProvider()})
	if err != nil {
		t.Fatalf("Observe() error = %v", err)
	}
	if !first.called || second.called {
		t.Fatalf("observer calls = first %v second %v, want copied map dispatch", first.called, second.called)
	}
	if result.ProviderID != "provider_cli" {
		t.Fatalf("ProviderID = %q, want provider_cli", result.ProviderID)
	}
	if result.Observations[0].ProviderID != "provider_cli" {
		t.Fatalf("observation provider = %q, want provider_cli", result.Observations[0].ProviderID)
	}
}

func TestObserveWrapsObserverFailures(t *testing.T) {
	observerErr := errors.New("capture failed")
	runner := NewRunner(map[model.ProviderKind]Observer{model.ProviderKindCLI: &fakeObserver{err: observerErr}})

	_, err := runner.Observe(context.Background(), &Request{Provider: cliProvider()})
	assertObserveCode(t, err, CodeObservationFailed)
	if !errors.Is(err, observerErr) {
		t.Fatalf("Observe() error = %v, want wrapped observer error", err)
	}
}

func TestObserveRejectsNilObserverResult(t *testing.T) {
	runner := NewRunner(map[model.ProviderKind]Observer{model.ProviderKindCLI: &fakeObserver{}})

	_, err := runner.Observe(context.Background(), &Request{Provider: cliProvider()})
	assertObserveCode(t, err, CodeObservationFailed)
}

func TestObserveStopsOnCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	observer := &fakeObserver{result: &Result{}}

	_, err := NewRunner(map[model.ProviderKind]Observer{model.ProviderKindCLI: observer}).Observe(ctx, &Request{Provider: cliProvider()})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Observe() error = %v, want context canceled", err)
	}
	if observer.called {
		t.Fatal("observer called with canceled context")
	}
}

func assertObserveCode(t *testing.T, err error, code string) {
	t.Helper()
	var observeErr *Error
	if !errors.As(err, &observeErr) {
		t.Fatalf("error = %v, want observe.Error code %s", err, code)
	}
	if observeErr.Code != code {
		t.Fatalf("code = %s, want %s", observeErr.Code, code)
	}
}

func cliProvider() *model.Provider {
	return &model.Provider{ID: "provider_cli", Kind: model.ProviderKindCLI, Path: "/tmp/provider"}
}

type fakeObserver struct {
	called bool
	result *Result
	err    error
}

func (observer *fakeObserver) Observe(context.Context, *Request) (*Result, error) {
	observer.called = true
	return observer.result, observer.err
}
