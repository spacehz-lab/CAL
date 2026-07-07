package contract

import (
	"encoding/json"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestContractConstants(t *testing.T) {
	if ErrorInvalidRequest != "invalid_request" || ErrorCaldUnavailable != "cald_unavailable" {
		t.Fatalf("error constants = %q/%q, want stable public values", ErrorInvalidRequest, ErrorCaldUnavailable)
	}
	if AcquisitionModeLive != "live" || AcquisitionModeReplay != "replay" || AcquisitionModeRules != "rules" {
		t.Fatalf("acquisition modes = %q/%q/%q, want stable public values", AcquisitionModeLive, AcquisitionModeReplay, AcquisitionModeRules)
	}
	if RunStrategyDefault != "default" || RunStrategyFirst != "first" || RunStrategyBest != "best" {
		t.Fatalf("run strategies = %q/%q/%q, want stable public values", RunStrategyDefault, RunStrategyFirst, RunStrategyBest)
	}
}

func TestAcquisitionRequestJSON(t *testing.T) {
	got := marshal(t, AcquisitionRequest{
		ProviderID:   "provider_cli",
		Hint:         "convert a document",
		ProposalPath: "/tmp/proposal.json",
		Mode:         AcquisitionModeReplay,
	})
	want := `{"provider_id":"provider_cli","hint":"convert a document","proposal_path":"/tmp/proposal.json","mode":"replay"}`
	if got != want {
		t.Fatalf("json = %s, want %s", got, want)
	}
}

func TestCapabilityListResponseJSON(t *testing.T) {
	got := marshal(t, CapabilityListResponse{
		Count: 1,
		Capabilities: []CapabilitySummary{{
			ID:          "document.convert",
			Description: "Convert a document.",
			Bindings: BindingSummary{
				Available:    1,
				ProviderIDs:  []string{"provider_cli"},
				VerifyLevels: []string{string(model.VerifyLevelL1)},
			},
		}},
	})
	want := `{"count":1,"capabilities":[{"id":"document.convert","description":"Convert a document.","bindings":{"available":1,"provider_ids":["provider_cli"],"verify_levels":["L1"]}}]}`
	if got != want {
		t.Fatalf("json = %s, want %s", got, want)
	}
}

func TestRunAndUseRequestJSON(t *testing.T) {
	runJSON := marshal(t, RunRequest{
		CapabilityID:   "document.convert",
		Inputs:         map[string]any{"target": "out.pdf"},
		Strategy:       RunStrategyFirst,
		Verify:         true,
		MinVerifyLevel: model.VerifyLevelL1,
	})
	wantRun := `{"capability_id":"document.convert","inputs":{"target":"out.pdf"},"strategy":"first","verify":true,"min_verify_level":"L1"}`
	if runJSON != wantRun {
		t.Fatalf("run json = %s, want %s", runJSON, wantRun)
	}

	useJSON := marshal(t, UseRequest{Intent: "convert to pdf", Strategy: RunStrategyBest})
	wantUse := `{"intent":"convert to pdf","strategy":"best"}`
	if useJSON != wantUse {
		t.Fatalf("use json = %s, want %s", useJSON, wantUse)
	}
}

func TestRunResponseJSON(t *testing.T) {
	got := marshal(t, RunResponse{Run: &model.Run{ID: "run_1", CapabilityID: "document.convert", Status: model.RunStatusSucceeded}})
	want := `{"run":{"id":"run_1","capability_id":"document.convert","status":"succeeded","verified":false}}`
	if got != want {
		t.Fatalf("json = %s, want %s", got, want)
	}
}

func TestErrorAndDaemonResponseJSON(t *testing.T) {
	errorJSON := marshal(t, ErrorResponse{Error: Error{Code: ErrorNotFound, Message: "not found"}})
	if want := `{"error":{"code":"not_found","message":"not found"}}`; errorJSON != want {
		t.Fatalf("error json = %s, want %s", errorJSON, want)
	}

	statusJSON := marshal(t, DaemonStatus{Running: true, BaseURL: "http://127.0.0.1:1234", PID: 42})
	if want := `{"running":true,"base_url":"http://127.0.0.1:1234","pid":42}`; statusJSON != want {
		t.Fatalf("status json = %s, want %s", statusJSON, want)
	}
}

func TestEvalResponseJSONDoesNotNeedEvalPackage(t *testing.T) {
	got := marshal(t, EvalResponse{
		Acquisition: AcquisitionMetrics{
			Traces:     CountByStatus{Total: 1, ByName: map[string]int{"completed": 1}},
			Candidates: 2,
			Probes:     ProbeMetrics{Total: 2, Passed: 1, Failed: 1},
			Promotions: PromotionMetrics{Total: 1, Capabilities: 1, Bindings: 1},
		},
		Reuse: ReuseMetrics{
			Runs:     CountByStatus{Total: 1, ByName: map[string]int{"succeeded": 1}},
			Verified: 1,
		},
		Capability: CapabilityMetrics{Capabilities: 1, Bindings: 1, PromotedBindings: 1, BindingsWithVerify: 1},
	})
	want := `{"acquisition":{"traces":{"total":1,"by_name":{"completed":1}},"candidates":2,"probes":{"total":2,"passed":1,"failed":1},"promotions":{"total":1,"capabilities":1,"bindings":1}},"reuse":{"runs":{"total":1,"by_name":{"succeeded":1}},"verified":1},"capability":{"capabilities":1,"bindings":1,"promoted_bindings":1,"bindings_with_verify":1,"capabilities_without_bindings":0}}`
	if got != want {
		t.Fatalf("json = %s, want %s", got, want)
	}
}

func marshal(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	return string(data)
}
