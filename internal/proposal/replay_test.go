package proposal

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

func TestParseReplayProposesResult(t *testing.T) {
	replay := mustParseReplay(t, replayJSON())

	result, err := replay.Propose(context.Background(), Request{
		Provider: core.Provider{ID: "provider_cli"},
	})
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}
	if len(result.Candidates) != 1 || len(result.ProbePlans) != 1 {
		t.Fatalf("result = %#v, want one candidate and one probe plan", result)
	}
	candidate := result.Candidates[0]
	if candidate.ProviderID != "provider_cli" || candidate.CapabilityID != "file.checksum" || candidate.Description == "" {
		t.Fatalf("candidate = %#v, want replay candidate", candidate)
	}
	plan := result.ProbePlans[0]
	if plan.CandidateIndex != 0 || plan.Verify.Level != core.VerifyLevelL2 || plan.Inputs["target"] != "/tmp/out.sha1" {
		t.Fatalf("probe plan = %#v, want replay probe plan", plan)
	}
}

func TestReplayFiltersProviderAndDebugFilter(t *testing.T) {
	replay := mustParseReplay(t, `{
		"candidates": [
			{
				"provider_id": "provider_other",
				"capability_id": "text.encode",
				"description": "Encode text.",
				"execution": {"kind": "cli", "spec": {"args": ["encode", "{{target}}"]}}
			},
			{
				"provider_id": "provider_cli",
				"capability_id": "file.checksum",
				"description": "Compute a checksum.",
				"execution": {"kind": "cli", "spec": {"args": ["sha1", "{{target}}"]}}
			}
		],
		"probe_plans": [
			{"candidate_index": 0, "inputs": {"target": "/tmp/encoded.txt"}, "verify": {"level":"L2","method":"execute","checks":[{"subject":{"type":"file","input":"target"},"predicate":"exists"}]}},
			{"candidate_index": 1, "inputs": {"target": "/tmp/out.sha1"}, "verify": {"level":"L2","method":"execute","checks":[{"subject":{"type":"file","input":"target"},"predicate":"exists"}]}}
		]
	}`)

	result, err := replay.Propose(context.Background(), Request{
		Provider:    core.Provider{ID: "provider_cli"},
		DebugFilter: "file.checksum",
	})
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}
	if len(result.Candidates) != 1 || result.Candidates[0].CapabilityID != "file.checksum" {
		t.Fatalf("candidates = %#v, want provider/debug filtered candidate", result.Candidates)
	}
	if len(result.ProbePlans) != 1 || result.ProbePlans[0].CandidateIndex != 0 {
		t.Fatalf("probe plans = %#v, want remapped candidate index", result.ProbePlans)
	}
}

func TestReplayRejectsOutOfRangeProbePlan(t *testing.T) {
	_, err := ParseReplay([]byte(`{
		"candidates": [{
			"capability_id": "file.checksum",
			"description": "Compute a checksum.",
			"execution": {"kind": "cli", "spec": {"args": ["sha1", "{{target}}"]}}
		}],
		"probe_plans": [{"candidate_index": 2, "verify": {"level":"L2","method":"execute","checks":[{"subject":{"type":"file","input":"target"},"predicate":"exists"}]}}]
	}`))
	if err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("ParseReplay() error = %v, want out of range error", err)
	}
}

func TestReplayRejectsMissingSelectedProbePlan(t *testing.T) {
	replay := mustParseReplay(t, `{
		"candidates": [
			{
				"provider_id": "provider_cli",
				"capability_id": "file.checksum",
				"description": "Compute a checksum.",
				"execution": {"kind": "cli", "spec": {"args": ["sha1", "{{target}}"]}}
			},
			{
				"provider_id": "provider_cli",
				"capability_id": "text.encode",
				"description": "Encode text.",
				"execution": {"kind": "cli", "spec": {"args": ["encode", "{{target}}"]}}
			}
		],
		"probe_plans": [
			{"candidate_index": 0, "inputs": {"target": "/tmp/out.sha1"}, "verify": {"level":"L2","method":"execute","checks":[{"subject":{"type":"file","input":"target"},"predicate":"exists"}]}}
		]
	}`)

	_, err := replay.Propose(context.Background(), Request{Provider: core.Provider{ID: "provider_cli"}})
	if err == nil || !strings.Contains(err.Error(), "has no probe plan") {
		t.Fatalf("Propose() error = %v, want missing probe plan error", err)
	}
}

func TestLoadReplayFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "proposal.json")
	if err := os.WriteFile(path, []byte(replayJSON()), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	replay, err := LoadReplayFile(path)
	if err != nil {
		t.Fatalf("LoadReplayFile() error = %v", err)
	}
	if len(replay.Candidates) != 1 || len(replay.ProbePlans) != 1 {
		t.Fatalf("replay = %#v, want loaded replay", replay)
	}
}

func mustParseReplay(t *testing.T, content string) Replay {
	t.Helper()
	replay, err := ParseReplay([]byte(content))
	if err != nil {
		t.Fatalf("ParseReplay() error = %v", err)
	}
	return replay
}

func replayJSON() string {
	return `{
		"candidates": [{
			"provider_id": "provider_cli",
			"capability_id": "file.checksum",
			"description": "Compute a file checksum.",
			"source": "replay:test",
			"execution": {
				"kind": "cli",
				"spec": {"args": ["sha1", "{{target}}"]}
			}
		}],
		"probe_plans": [{
			"candidate_index": 0,
			"inputs": {"target": "/tmp/out.sha1"},
			"verify": {"level":"L2","method":"execute","checks":[{"subject":{"type":"file","input":"target"},"predicate":"exists"}]}
		}]
	}`
}
