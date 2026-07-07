package e2e

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/contract"
	"github.com/spacehz-lab/cal/internal/model"
)

func assertAcquisitionCompleted(t *testing.T, response *contract.AcquisitionResponse, capabilityPromotions int, bindingPromotions int) model.Trace {
	t.Helper()
	if response == nil {
		t.Fatal("acquisition response is nil")
	}
	if response.TraceID == "" || response.Trace == nil || response.Trace.Status != model.TraceStatusCompleted || response.Error != nil {
		t.Fatalf("acquisition = %#v, want completed trace", response)
	}
	if response.CapabilitiesPromoted != capabilityPromotions || response.BindingsPromoted != bindingPromotions {
		t.Fatalf("acquisition promoted = %d/%d, want %d/%d", response.CapabilitiesPromoted, response.BindingsPromoted, capabilityPromotions, bindingPromotions)
	}
	return *response.Trace
}

func assertTraceStored(t *testing.T, home string, traceID string) model.Trace {
	t.Helper()
	trace := readJSONFile[model.Trace](t, filepath.Join(home, "traces", traceID, "trace.json"))
	if trace.ID != traceID || trace.Status != model.TraceStatusCompleted {
		t.Fatalf("stored trace = %#v, want completed trace %s", trace, traceID)
	}
	return trace
}

func assertLiveLLMTrace(t *testing.T, trace model.Trace) string {
	t.Helper()
	if len(trace.Candidates) != 1 {
		t.Fatalf("trace candidates = %#v, want one candidate", trace.Candidates)
	}
	candidate := trace.Candidates[0]
	if candidate.CapabilityID == "" || candidate.Description == "" {
		t.Fatalf("candidate = %#v, want capability id and description", candidate)
	}
	args := executionArgs(candidate.Execution)
	wantArgs := []string{"make-pdf", "--in", "{{source}}", "--out", "{{target}}"}
	if len(args) != len(wantArgs) {
		t.Fatalf("candidate args = %#v, want %#v", args, wantArgs)
	}
	for index, want := range wantArgs {
		if args[index] != want {
			t.Fatalf("candidate args[%d] = %#v, want %q", index, args[index], want)
		}
	}
	assertTraceVerifyForCapability(t, trace, candidate.CapabilityID)
	promotions := trace.Promotions
	if len(promotions) != 1 || promotions[0].CapabilityID != candidate.CapabilityID || promotions[0].BindingID == "" {
		t.Fatalf("trace promotions = %#v, want %s promotion", promotions, candidate.CapabilityID)
	}
	return candidate.CapabilityID
}

func assertLiveLLMMultiCapTrace(t *testing.T, trace model.Trace) (string, string) {
	t.Helper()
	if len(trace.Candidates) != 2 || len(trace.Probes) != 2 || len(trace.Promotions) != 2 {
		t.Fatalf("trace = %#v, want two candidates, probes, and promotions", trace)
	}
	var pdfCapabilityID string
	var noteCapabilityID string
	for _, candidate := range trace.Candidates {
		if candidate.CapabilityID == "" || candidate.Description == "" {
			t.Fatalf("candidate = %#v, want capability id and description", candidate)
		}
		args := executionArgs(candidate.Execution)
		if len(args) == 0 {
			t.Fatalf("candidate args = %#v, want non-empty args", candidate.Execution.Spec[model.ExecutionSpecArgs])
		}
		switch args[0] {
		case "make-pdf":
			pdfCapabilityID = candidate.CapabilityID
		case "write-note":
			noteCapabilityID = candidate.CapabilityID
		default:
			t.Fatalf("candidate = %#v, want make-pdf or write-note execution", candidate)
		}
	}
	if pdfCapabilityID == "" || noteCapabilityID == "" || pdfCapabilityID == noteCapabilityID {
		t.Fatalf("trace candidates = %#v, want distinct PDF and note capabilities", trace.Candidates)
	}
	for _, promotion := range trace.Promotions {
		if promotion.CandidateIndex < 0 || promotion.CandidateIndex >= len(trace.Candidates) {
			t.Fatalf("promotion = %#v, want valid candidate index", promotion)
		}
		capabilityID := trace.Candidates[promotion.CandidateIndex].CapabilityID
		if promotion.CapabilityID != capabilityID || promotion.CapabilityAction != "created" || promotion.BindingAction != "created" || promotion.BindingID == "" {
			t.Fatalf("promotion = %#v, want created promotion for %s", promotion, capabilityID)
		}
	}
	return pdfCapabilityID, noteCapabilityID
}

func assertTraceVerifyForCapability(t *testing.T, trace model.Trace, capabilityID string) {
	t.Helper()
	for _, probe := range trace.Probes {
		if probe.CandidateIndex < 0 || probe.CandidateIndex >= len(trace.Candidates) {
			continue
		}
		if trace.Candidates[probe.CandidateIndex].CapabilityID == capabilityID && probe.Passed && model.VerifyLevelRank(probe.Verify.Level) >= model.VerifyLevelRank(model.VerifyLevelL1) {
			return
		}
	}
	t.Fatalf("trace probes = %#v, missing passing verify for %s", trace.Probes, capabilityID)
}

func assertLiveLLMCapabilityProbe(t *testing.T, trace model.Trace, capabilityID string, level model.VerifyLevel, method model.VerifyMethod) {
	t.Helper()
	for _, probe := range trace.Probes {
		if probe.CandidateIndex < 0 || probe.CandidateIndex >= len(trace.Candidates) {
			continue
		}
		if trace.Candidates[probe.CandidateIndex].CapabilityID != capabilityID {
			continue
		}
		if !probe.Passed || probe.Verify.Level != level || probe.Verify.Method != method {
			t.Fatalf("probe for %s = %#v, want passed %s %s", capabilityID, probe, level, method)
		}
		if len(probe.Evidence) == 0 {
			t.Fatalf("probe for %s evidence = %#v, want evidence", capabilityID, probe.Evidence)
		}
		return
	}
	t.Fatalf("trace probes = %#v, missing probe for %s", trace.Probes, capabilityID)
}

func assertLiveLLMCapabilityProbeAtLeast(t *testing.T, trace model.Trace, capabilityID string, level model.VerifyLevel, method model.VerifyMethod) {
	t.Helper()
	for _, probe := range trace.Probes {
		if probe.CandidateIndex < 0 || probe.CandidateIndex >= len(trace.Candidates) {
			continue
		}
		if trace.Candidates[probe.CandidateIndex].CapabilityID != capabilityID {
			continue
		}
		if !probe.Passed || model.VerifyLevelRank(probe.Verify.Level) < model.VerifyLevelRank(level) || probe.Verify.Method != method {
			t.Fatalf("probe for %s = %#v, want passed %s+ %s", capabilityID, probe, level, method)
		}
		if len(probe.Evidence) == 0 {
			t.Fatalf("probe for %s evidence = %#v, want evidence", capabilityID, probe.Evidence)
		}
		return
	}
	t.Fatalf("trace probes = %#v, missing probe for %s", trace.Probes, capabilityID)
}

func liveLLMCapabilityIDForExecution(t *testing.T, trace model.Trace, method model.VerifyMethod, argsPrefix ...string) string {
	t.Helper()
	for index, candidate := range trace.Candidates {
		args := executionArgs(candidate.Execution)
		if !hasArgsPrefix(args, argsPrefix) {
			continue
		}
		if !traceHasPassedProbe(t, trace, index, method) {
			continue
		}
		if candidate.CapabilityID == "" {
			t.Fatalf("candidate = %#v, want capability id", candidate)
		}
		return candidate.CapabilityID
	}
	t.Fatalf("trace candidates = %#v, missing passed %s candidate with args prefix %#v", trace.Candidates, method, argsPrefix)
	return ""
}

func liveLLMOutputPathInput(t *testing.T, trace model.Trace, capabilityID string) string {
	t.Helper()
	for _, probe := range trace.Probes {
		if probe.CandidateIndex < 0 || probe.CandidateIndex >= len(trace.Candidates) {
			continue
		}
		if trace.Candidates[probe.CandidateIndex].CapabilityID != capabilityID || !probe.Passed || probe.Verify.Method != model.VerifyMethodExecute {
			continue
		}
		for _, check := range probe.Verify.Checks {
			if check.Subject.Type == model.VerifySubjectFile && strings.TrimSpace(check.Subject.Input) != "" {
				return check.Subject.Input
			}
		}
	}
	t.Fatalf("trace probes = %#v, missing execute file output input for %s", trace.Probes, capabilityID)
	return ""
}

func traceHasPassedProbe(t *testing.T, trace model.Trace, candidateIndex int, method model.VerifyMethod) bool {
	t.Helper()
	for _, probe := range trace.Probes {
		if probe.CandidateIndex == candidateIndex && probe.Passed && probe.Verify.Method == method {
			return true
		}
	}
	return false
}

func assertRunSucceeded(t *testing.T, response contract.RunResponse) *model.Run {
	t.Helper()
	if response.Run == nil || response.Run.Status != model.RunStatusSucceeded {
		t.Fatalf("run response = %#v, want succeeded run", response)
	}
	return response.Run
}

func hasArgsPrefix(args []string, prefix []string) bool {
	if len(args) < len(prefix) {
		return false
	}
	for index, want := range prefix {
		if args[index] != want {
			return false
		}
	}
	return true
}

func executionArgs(execution model.Execution) []string {
	value, ok := execution.Spec[model.ExecutionSpecArgs]
	if !ok {
		return nil
	}
	switch args := value.(type) {
	case []string:
		return append([]string(nil), args...)
	case []any:
		out := make([]string, 0, len(args))
		for _, arg := range args {
			text, ok := arg.(string)
			if ok {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}
