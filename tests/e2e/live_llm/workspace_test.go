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
		if t.Failed() || os.Getenv("CAL_LIVE_LLM_KEEP_HOME") == "1" {
			logLiveLLMWorkspace(t, workspace)
			return
		}
		_ = os.RemoveAll(root)
	})
	return workspace
}

func logLiveLLMWorkspace(t *testing.T, workspace liveLLMWorkspaceDir) {
	t.Helper()
	t.Logf("live LLM workspace retained: %s", workspace.root)
	t.Logf("live LLM CAL_HOME retained: %s", workspace.home)
	logLiveLLMTraces(t, workspace.home)
}

func logLiveLLMTraces(t *testing.T, home string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(home, "discovery"))
	if err != nil {
		t.Logf("live LLM discovery traces unavailable: %v", err)
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(home, "discovery", entry.Name(), "trace.json")
		trace := e2etest.ReadJSONFile[caltrace.Trace](t, path)
		t.Logf("live LLM trace: id=%s status=%s path=%s error=%s", trace.ID, trace.Status, path, traceErrorMessage(trace))
		if trace.Proposal == nil {
			continue
		}
		for _, stage := range trace.Proposal.Stages {
			t.Logf("live LLM proposal stage: trace=%s stage=%s summary=%s", trace.ID, stage.Name, jsonString(stage.Summary))
			for _, item := range stage.Items {
				if item.Decision != caltrace.ProposalDecisionKeep || item.Reason != "" {
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

func traceErrorMessage(trace caltrace.Trace) string {
	return recordErrorMessage(trace.Error)
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
