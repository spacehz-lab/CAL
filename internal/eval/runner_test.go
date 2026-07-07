package eval

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestRunReturnsZeroMetricsForEmptyStore(t *testing.T) {
	result, err := NewRunner(&fakeStore{}).Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Acquisition.Traces.Total != 0 || result.Reuse.Runs.Total != 0 || result.Capability.Capabilities != 0 {
		t.Fatalf("result = %#v, want zero metrics", result)
	}
}

func TestRunAggregatesMetrics(t *testing.T) {
	store := &fakeStore{
		traces: []model.Trace{{
			ID:     "trace_1",
			Status: model.TraceStatusCompleted,
			Candidates: []model.Candidate{
				{ProviderID: "provider_a", CapabilityID: "capability_a"},
				{ProviderID: "provider_b", CapabilityID: "capability_b"},
			},
			Probes: []model.Probe{
				{CandidateIndex: 0, Passed: true},
				{CandidateIndex: 1, Passed: false, Error: &model.RecordError{Code: "probe_failed"}},
			},
			Promotions: []model.Promotion{
				{CapabilityID: "capability_a", BindingID: "binding_a", ProviderID: "provider_a"},
				{CapabilityID: "capability_a", BindingID: "binding_a", ProviderID: "provider_a"},
			},
		}, {
			ID:     "trace_2",
			Status: model.TraceStatusFailed,
			Error:  &model.RecordError{Code: "observe_failed"},
		}},
		runs: []model.Run{{
			ID:           "run_1",
			CapabilityID: "capability_a",
			ProviderID:   "provider_a",
			Status:       model.RunStatusSucceeded,
			Verified:     true,
		}, {
			ID:           "run_2",
			CapabilityID: "capability_b",
			ProviderID:   "provider_b",
			Status:       model.RunStatusFailed,
			Error:        &model.RecordError{Code: "execution_failed"},
		}},
		capabilities: []model.Capability{
			capability("capability_a", binding("binding_a", "provider_a", true)),
			{ID: "capability_empty"},
		},
	}

	result, err := NewRunner(store).Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Acquisition.Traces.Total != 2 || result.Acquisition.Traces.ByName[string(model.TraceStatusCompleted)] != 1 || result.Acquisition.Traces.ByName[string(model.TraceStatusFailed)] != 1 {
		t.Fatalf("acquisition traces = %#v, want completed and failed counts", result.Acquisition.Traces)
	}
	if result.Acquisition.Candidates != 2 || result.Acquisition.Probes.Total != 2 || result.Acquisition.Probes.Passed != 1 || result.Acquisition.Probes.Failed != 1 {
		t.Fatalf("acquisition metrics = %#v, want candidates/probes", result.Acquisition)
	}
	if result.Acquisition.Promotions.Total != 2 || result.Acquisition.Promotions.Capabilities != 1 || result.Acquisition.Promotions.Bindings != 1 {
		t.Fatalf("promotions = %#v, want total 2 and unique promoted ids", result.Acquisition.Promotions)
	}
	if result.Acquisition.Errors["probe_failed"] != 1 || result.Acquisition.Errors["observe_failed"] != 1 {
		t.Fatalf("acquisition errors = %#v, want probe and trace errors", result.Acquisition.Errors)
	}
	if result.Reuse.Runs.Total != 2 || result.Reuse.Runs.ByName[string(model.RunStatusSucceeded)] != 1 || result.Reuse.Runs.ByName[string(model.RunStatusFailed)] != 1 {
		t.Fatalf("reuse runs = %#v, want success and failure counts", result.Reuse.Runs)
	}
	if result.Reuse.Verified != 1 || result.Reuse.Errors["execution_failed"] != 1 {
		t.Fatalf("reuse metrics = %#v, want verified and error counts", result.Reuse)
	}
	if result.Reuse.ByProvider["provider_a"].Verified != 1 || result.Reuse.ByCapability["capability_b"].Runs.ByName[string(model.RunStatusFailed)] != 1 {
		t.Fatalf("reuse groups = %#v %#v, want grouped runs", result.Reuse.ByProvider, result.Reuse.ByCapability)
	}
	if result.Capability.Capabilities != 2 || result.Capability.Bindings != 1 || result.Capability.PromotedBindings != 1 || result.Capability.BindingsWithVerify != 1 || result.Capability.CapabilitiesWithoutBindings != 1 {
		t.Fatalf("capability metrics = %#v, want coverage counts", result.Capability)
	}
}

func TestRunFiltersByProviderAndCapability(t *testing.T) {
	store := &fakeStore{
		traces: []model.Trace{{
			ID:     "trace_a",
			Status: model.TraceStatusCompleted,
			Candidates: []model.Candidate{
				{ProviderID: "provider_a", CapabilityID: "capability_a"},
			},
		}, {
			ID:     "trace_b",
			Status: model.TraceStatusCompleted,
			Candidates: []model.Candidate{
				{ProviderID: "provider_b", CapabilityID: "capability_b"},
			},
		}},
		runs: []model.Run{{
			ID:           "run_a",
			CapabilityID: "capability_a",
			ProviderID:   "provider_a",
			Status:       model.RunStatusSucceeded,
		}, {
			ID:           "run_b",
			CapabilityID: "capability_b",
			ProviderID:   "provider_b",
			Status:       model.RunStatusSucceeded,
		}},
		capabilities: []model.Capability{
			capability("capability_a", binding("binding_a", "provider_a", true)),
			capability("capability_b", binding("binding_b", "provider_b", true)),
		},
	}

	result, err := NewRunner(store).Run(context.Background(), &Request{ProviderID: "provider_a", CapabilityID: "capability_a"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Acquisition.Traces.Total != 1 || result.Reuse.Runs.Total != 1 || result.Capability.Capabilities != 1 {
		t.Fatalf("result = %#v, want only provider_a/capability_a records", result)
	}
	if result.Reuse.ByProvider["provider_b"].Runs.Total != 0 {
		t.Fatalf("by provider = %#v, want provider_b filtered out", result.Reuse.ByProvider)
	}
}

func TestRunReturnsStoreErrors(t *testing.T) {
	tests := []struct {
		name string
		errs fakeStore
		want string
	}{
		{name: "traces", errs: fakeStore{traceErr: errors.New("trace failed")}, want: "list traces"},
		{name: "runs", errs: fakeStore{runErr: errors.New("run failed")}, want: "list runs"},
		{name: "capabilities", errs: fakeStore{capabilityErr: errors.New("capability failed")}, want: "list capabilities"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewRunner(&test.errs).Run(context.Background(), nil)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Run() error = %v, want %s error", err, test.want)
			}
		})
	}
}

func TestRunStopsWhenContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewRunner(&fakeStore{}).Run(ctx, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want context canceled", err)
	}
}

func TestRunRequiresStore(t *testing.T) {
	_, err := NewRunner(nil).Run(context.Background(), nil)
	if err == nil {
		t.Fatal("Run() error = nil, want missing store error")
	}
}

type fakeStore struct {
	traces        []model.Trace
	runs          []model.Run
	capabilities  []model.Capability
	traceErr      error
	runErr        error
	capabilityErr error
}

func (store *fakeStore) ListTraces() ([]model.Trace, error) {
	if store.traceErr != nil {
		return nil, store.traceErr
	}
	return store.traces, nil
}

func (store *fakeStore) ListRuns() ([]model.Run, error) {
	if store.runErr != nil {
		return nil, store.runErr
	}
	return store.runs, nil
}

func (store *fakeStore) ListCapabilities() ([]model.Capability, error) {
	if store.capabilityErr != nil {
		return nil, store.capabilityErr
	}
	return store.capabilities, nil
}

func capability(id string, bindings ...model.Binding) model.Capability {
	for index := range bindings {
		bindings[index].CapabilityID = id
	}
	return model.Capability{ID: id, Description: "test capability", Bindings: bindings}
}

func binding(id string, providerID string, withVerify bool) model.Binding {
	binding := model.Binding{
		ID:         id,
		ProviderID: providerID,
		Execution:  model.Execution{Kind: model.ExecutionKindCLI},
		Evidence:   []model.EvidenceRef{{ID: "evidence_" + id}},
		State:      model.BindingStatePromoted,
	}
	if withVerify {
		binding.Verify = &model.VerifySpec{
			Level:  model.VerifyLevelL2,
			Method: model.VerifyMethodExecute,
		}
	}
	return binding
}
