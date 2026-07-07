package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

const (
	envLiveLLME2E      = "CAL_LIVE_LLM_E2E"
	envLiveLLMKeepHome = "CAL_LIVE_LLM_KEEP_HOME"
	envLLMAPI          = "CAL_LLM_API"
	envLLMModel        = "CAL_LLM_MODEL"
	envLLMAPIKey       = "CAL_LLM_API_KEY"
)

type liveLLMWorkspaceDir struct {
	root string
	home string
	temp string
}

func liveLLMWorkspace(t *testing.T) liveLLMWorkspaceDir {
	t.Helper()
	root, err := os.MkdirTemp("", t.Name())
	if err != nil {
		t.Fatalf("create live llm workspace: %v", err)
	}
	workspace := liveLLMWorkspaceDir{
		root: root,
		home: filepath.Join(root, "home"),
		temp: filepath.Join(root, "work"),
	}
	if err := os.MkdirAll(workspace.temp, 0o755); err != nil {
		t.Fatalf("create live llm work dir: %v", err)
	}
	t.Cleanup(func() {
		if t.Failed() || os.Getenv(envLiveLLMKeepHome) == "1" {
			logLiveLLMWorkspace(t, workspace)
			return
		}
		_ = os.RemoveAll(root)
	})
	return workspace
}

func liveLLMEnv(t *testing.T, home string) []string {
	t.Helper()
	if !liveLLMEnabled() {
		t.Skip("set CAL_LIVE_LLM_E2E=1 and CAL_LLM_* to run live LLM e2e")
	}
	required := []string{envLLMAPI, envLLMModel, envLLMAPIKey}
	for _, name := range required {
		if strings.TrimSpace(os.Getenv(name)) == "" {
			t.Skipf("set %s to run live LLM e2e", name)
		}
	}
	if os.Getenv(envLLMAPI) != "chat_completions" {
		t.Skip("live LLM e2e requires CAL_LLM_API=chat_completions")
	}
	return withHomeEnv(os.Environ(), home)
}

func liveLLMEnabled() bool {
	return os.Getenv(envLiveLLME2E) == "1"
}

func logLiveLLMWorkspace(t *testing.T, workspace liveLLMWorkspaceDir) {
	t.Helper()
	t.Logf("live LLM workspace retained: %s", workspace.root)
	t.Logf("live LLM CAL_HOME retained: %s", workspace.home)
	logLiveLLMTraces(t, workspace.home)
}

func logLiveLLMTraces(t *testing.T, home string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(home, "traces"))
	if err != nil {
		t.Logf("live LLM traces unavailable: %v", err)
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(home, "traces", entry.Name(), "trace.json")
		trace := readJSONFile[model.Trace](t, path)
		t.Logf("live LLM trace: id=%s status=%s path=%s error=%s", trace.ID, trace.Status, path, recordErrorMessage(trace.Error))
		if trace.Proposal == nil {
			continue
		}
		for _, stage := range trace.Proposal.Stages {
			t.Logf("live LLM proposal stage: trace=%s stage=%s summary=%s", trace.ID, stage.Name, jsonString(stage.Summary))
			for _, item := range stage.Items {
				if item.Decision != model.ProposalDecisionKeep || item.Reason != "" {
					t.Logf("live LLM proposal item: trace=%s stage=%s id=%s name=%q decision=%s reason=%s", trace.ID, stage.Name, item.ID, item.Name, item.Decision, item.Reason)
				}
			}
		}
		for _, attempt := range trace.Proposal.Attempts {
			t.Logf("live LLM proposal attempt: trace=%s stage=%s capability=%s candidate_index=%s status=%s duration_ms=%d error=%s raw=%s",
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
