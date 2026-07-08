package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/contract"
	"github.com/spacehz-lab/cal/internal/model"
)

func assertAcquisitionCompleted(t *testing.T, response contract.AcquisitionResponse) model.Trace {
	t.Helper()
	if response.TraceID == "" || response.Trace == nil || response.Trace.Status != model.TraceStatusCompleted || response.Error != nil {
		t.Fatalf("acquisition = %#v, want completed trace", response)
	}
	if response.CapabilitiesPromoted == 0 || response.BindingsPromoted == 0 {
		t.Fatalf("acquisition promoted = %d/%d, want at least one capability and binding", response.CapabilitiesPromoted, response.BindingsPromoted)
	}
	return *response.Trace
}

func assertCanaryProbe(t *testing.T, trace model.Trace, minLevel model.VerifyLevel, method model.VerifyMethod, checks ...checkMatcher) string {
	t.Helper()
	for _, probe := range trace.Probes {
		if probe.CandidateIndex < 0 || probe.CandidateIndex >= len(trace.Candidates) {
			continue
		}
		if !probe.Passed || probe.Verify.Method != method || model.VerifyLevelRank(probe.Verify.Level) < model.VerifyLevelRank(minLevel) {
			continue
		}
		if !probeHasChecks(probe, checks) {
			continue
		}
		capabilityID := trace.Candidates[probe.CandidateIndex].CapabilityID
		if strings.TrimSpace(capabilityID) == "" {
			t.Fatalf("candidate = %#v, want capability id", trace.Candidates[probe.CandidateIndex])
		}
		return capabilityID
	}
	t.Fatalf("trace probes = %#v, missing passed %s+ %s probe", trace.Probes, minLevel, method)
	return ""
}

func assertRunSucceeded(t *testing.T, response contract.RunResponse, capabilityID string, providerID string) *model.Run {
	t.Helper()
	if response.Run == nil {
		t.Fatalf("run response = %#v, want run", response)
	}
	if response.Run.Status != model.RunStatusSucceeded || !response.Run.Verified || response.Run.CapabilityID != capabilityID || response.Run.ProviderID != providerID {
		t.Fatalf("run = %#v, want verified successful %s run on provider %s", response.Run, capabilityID, providerID)
	}
	if len(response.Run.Evidence) == 0 {
		t.Fatalf("run evidence = %#v, want verification evidence", response.Run.Evidence)
	}
	return response.Run
}

func readJSONFile[T any](t *testing.T, path string) T {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var value T
	if err := json.Unmarshal(content, &value); err != nil {
		t.Fatalf("decode %s: %v\n%s", path, err, content)
	}
	return value
}

func jsonInputs(t *testing.T, values map[string]any) string {
	t.Helper()
	content, err := json.Marshal(values)
	if err != nil {
		t.Fatalf("marshal inputs: %v", err)
	}
	return string(content)
}

func logCliCanaryWorkspace(t *testing.T, workspace cliCanaryWorkspace) {
	t.Helper()
	t.Logf("CLI canary workspace retained: %s", workspace.root)
	t.Logf("CLI canary CAL_HOME retained: %s", workspace.home)
	logCliCanaryTraces(t, workspace.home)
}

func logCliCanaryTraces(t *testing.T, home string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(home, "traces"))
	if err != nil {
		t.Logf("CLI canary traces unavailable: %v", err)
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(home, "traces", entry.Name(), "trace.json")
		trace := readJSONFile[model.Trace](t, path)
		t.Logf("CLI canary trace: id=%s status=%s path=%s error=%s", trace.ID, trace.Status, path, recordErrorMessage(trace.Error))
		if trace.Proposal == nil {
			continue
		}
		for _, stage := range trace.Proposal.Stages {
			t.Logf("CLI canary proposal stage: trace=%s stage=%s summary=%s", trace.ID, stage.Name, jsonString(stage.Summary))
		}
		for _, attempt := range trace.Proposal.Attempts {
			t.Logf("CLI canary proposal attempt: trace=%s stage=%s capability=%s candidate_index=%s status=%s duration_ms=%d error=%s raw=%s",
				trace.ID,
				attempt.Stage,
				attempt.CapabilityID,
				candidateIndexString(attempt.CandidateIndex),
				attempt.Status,
				attempt.DurationMS,
				recordErrorMessage(attempt.Error),
				attempt.RawResponse,
			)
		}
	}
}

type checkMatcher struct {
	subject   model.VerifySubjectType
	predicate model.VerifyPredicate
}

func check(subject model.VerifySubjectType, predicate model.VerifyPredicate) checkMatcher {
	return checkMatcher{subject: subject, predicate: predicate}
}

func probeHasChecks(probe model.Probe, checks []checkMatcher) bool {
	for _, want := range checks {
		var found bool
		for _, got := range probe.Verify.Checks {
			if got.Subject.Type == want.subject && got.Predicate == want.predicate {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func recordErrorMessage(err *model.RecordError) string {
	if err == nil {
		return ""
	}
	return err.Code + ": " + err.Message
}

func candidateIndexString(index *int) string {
	if index == nil {
		return ""
	}
	data, _ := json.Marshal(*index)
	return string(data)
}

func jsonString(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(data)
}
