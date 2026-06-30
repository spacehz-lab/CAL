package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
	e2etest "github.com/spacehz-lab/cal/internal/testsupport/e2e"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

type cliCanaryWorkspaceDir struct {
	root string
	home string
	temp string
}

type cliCanaryHarness struct {
	repo      string
	env       []string
	calctlBin string
	home      string
	temp      string
}

type acquisitionOutput struct {
	State                string                    `json:"state"`
	TraceID              string                    `json:"trace_id"`
	CapabilitiesPromoted int                       `json:"capabilities_promoted"`
	BindingsPromoted     int                       `json:"bindings_promoted"`
	Providers            []e2etest.ProviderSummary `json:"providers"`
}

type runOutput struct {
	Status   string             `json:"status"`
	Verified bool               `json:"verified"`
	Evidence []core.EvidenceRef `json:"evidence"`
	Outputs  map[string]any     `json:"outputs"`
	Error    *core.RecordError  `json:"error,omitempty"`
}

func newCliCanaryHarness(t *testing.T) cliCanaryHarness {
	t.Helper()
	workspace := cliCanaryWorkspace(t)
	env := cliCanaryEnv(t, workspace.home)
	repo := e2etest.RepoRoot(t)
	calctlBin := filepath.Join(workspace.temp, "calctl")
	caldBin := filepath.Join(workspace.temp, "cald")
	e2etest.Build(t, repo, calctlBin, "./cmd/calctl")
	e2etest.Build(t, repo, caldBin, "./cmd/cald")
	e2etest.StartCald(t, repo, env, caldBin)
	return cliCanaryHarness{
		repo:      repo,
		env:       env,
		calctlBin: calctlBin,
		home:      workspace.home,
		temp:      workspace.temp,
	}
}

func cliCanaryWorkspace(t *testing.T) cliCanaryWorkspaceDir {
	t.Helper()
	root, err := os.MkdirTemp("", t.Name())
	if err != nil {
		t.Fatalf("create cli canary workspace: %v", err)
	}
	workspace := cliCanaryWorkspaceDir{
		root: root,
		home: filepath.Join(root, "home"),
		temp: filepath.Join(root, "work"),
	}
	if err := os.MkdirAll(workspace.temp, 0o755); err != nil {
		t.Fatalf("create cli canary work dir: %v", err)
	}
	t.Cleanup(func() {
		if t.Failed() || os.Getenv("CAL_CLI_CANARY_LLM_KEEP_HOME") == "1" {
			logCliCanaryWorkspace(t, workspace)
			return
		}
		_ = os.RemoveAll(root)
	})
	return workspace
}

func cliCanaryEnv(t *testing.T, home string) []string {
	t.Helper()
	if os.Getenv("CAL_CLI_CANARY_LLM_E2E") != "1" {
		t.Skip("set CAL_CLI_CANARY_LLM_E2E=1 and CAL_LLM_* to run CLI canary LLM e2e")
	}
	required := []string{"CAL_LLM_API", "CAL_LLM_MODEL", "CAL_LLM_API_KEY"}
	for _, name := range required {
		if os.Getenv(name) == "" {
			t.Skipf("set %s to run CLI canary LLM e2e", name)
		}
	}
	if os.Getenv("CAL_LLM_API") != "chat_completions" {
		t.Skip("CLI canary LLM e2e requires CAL_LLM_API=chat_completions")
	}
	return e2etest.WithHomeEnv(os.Environ(), home)
}

func (h cliCanaryHarness) discover(t *testing.T, providerPath string, capabilityID string) (e2etest.ProviderSummary, acquisitionOutput, caltrace.Trace) {
	t.Helper()
	var provider e2etest.ProviderSummary
	e2etest.RunJSON(t, h.repo, h.env, &provider, h.calctlBin, "providers", "add", "--provider-path", providerPath, "--json")

	var acquisition acquisitionOutput
	e2etest.RunJSON(t, h.repo, h.env, &acquisition, h.calctlBin, "discovery", "run", "--provider-id", provider.ID, "--capability-id", capabilityID, "--json")
	if acquisition.State != "succeeded" || acquisition.TraceID == "" || acquisition.BindingsPromoted == 0 {
		t.Fatalf("acquisition for %s = %#v, want promoted binding", capabilityID, acquisition)
	}
	trace := e2etest.ReadJSONFile[caltrace.Trace](t, filepath.Join(h.home, "discovery", acquisition.TraceID, "trace.json"))
	return provider, acquisition, trace
}

func (h cliCanaryHarness) run(t *testing.T, providerID, capabilityID string, inputs map[string]any, verify bool, args ...string) runOutput {
	t.Helper()
	command := []string{"runs", "create", "--capability-id", capabilityID, "--provider-id", providerID, "--inputs-json", e2etest.JSONArg(t, inputs)}
	if verify {
		command = append(command, "--verify")
	}
	command = append(command, args...)
	command = append(command, "--json")
	var output runOutput
	e2etest.RunJSON(t, h.repo, h.env, &output, h.calctlBin, command...)
	return output
}

func (h cliCanaryHarness) runFail(t *testing.T, providerID, capabilityID string, inputs map[string]any, args ...string) runOutput {
	t.Helper()
	command := []string{"runs", "create", "--capability-id", capabilityID, "--provider-id", providerID, "--inputs-json", e2etest.JSONArg(t, inputs)}
	command = append(command, args...)
	command = append(command, "--json")
	var output runOutput
	e2etest.RunFailJSON(t, h.repo, h.env, &output, h.calctlBin, command...)
	return output
}

func assertCliCanaryProbe(t *testing.T, trace caltrace.Trace, capabilityID string, minLevel core.VerifyLevel, method core.VerifyMethod) caltrace.Probe {
	t.Helper()
	for _, probe := range trace.Probes {
		if probe.CandidateIndex < 0 || probe.CandidateIndex >= len(trace.Candidates) {
			continue
		}
		if trace.Candidates[probe.CandidateIndex].CapabilityID != capabilityID {
			continue
		}
		if !probe.Passed || core.VerifyLevelRank(probe.Verify.Level) < core.VerifyLevelRank(minLevel) || probe.Verify.Method != method {
			t.Fatalf("probe for %s = %#v, want passed %s+ %s", capabilityID, probe, minLevel, method)
		}
		return probe
	}
	t.Fatalf("trace probes = %#v, missing probe for %s", trace.Probes, capabilityID)
	return caltrace.Probe{}
}

func assertCliCanaryCheck(t *testing.T, probe caltrace.Probe, subject core.VerifySubjectType, predicate core.VerifyPredicate) {
	t.Helper()
	for _, check := range probe.Verify.Checks {
		if check.Subject.Type == subject && check.Predicate == predicate {
			return
		}
	}
	t.Fatalf("probe verify = %#v, want %s %s check", probe.Verify, subject, predicate)
}

func assertCliCanaryCandidateArg(t *testing.T, trace caltrace.Trace, capabilityID string, want string) {
	t.Helper()
	for _, candidate := range trace.Candidates {
		if candidate.CapabilityID != capabilityID {
			continue
		}
		args, ok := candidate.Execution.Spec[core.ExecutionSpecArgs].([]any)
		if !ok {
			t.Fatalf("candidate %s args = %#v, want JSON array", capabilityID, candidate.Execution.Spec[core.ExecutionSpecArgs])
		}
		for _, arg := range args {
			if arg == want {
				return
			}
		}
		t.Fatalf("candidate %s args = %#v, want arg %q", capabilityID, args, want)
	}
	t.Fatalf("trace candidates = %#v, missing candidate for %s", trace.Candidates, capabilityID)
}

func logCliCanaryWorkspace(t *testing.T, workspace cliCanaryWorkspaceDir) {
	t.Helper()
	t.Logf("CLI canary workspace retained: %s", workspace.root)
	t.Logf("CLI canary CAL_HOME retained: %s", workspace.home)
	entries, err := os.ReadDir(filepath.Join(workspace.home, "discovery"))
	if err != nil {
		t.Logf("CLI canary discovery traces unavailable: %v", err)
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(workspace.home, "discovery", entry.Name(), "trace.json")
		trace := e2etest.ReadJSONFile[caltrace.Trace](t, path)
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

func recordErrorMessage(err *core.RecordError) string {
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
