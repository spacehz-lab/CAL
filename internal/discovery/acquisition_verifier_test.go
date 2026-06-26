package discovery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/proposal"
	"github.com/spacehz-lab/cal/internal/runtime"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestAcquisitionVerifierReportsProbePlanFailure(t *testing.T) {
	now := time.Unix(10, 0).UTC()
	plannerErr := errors.New("cannot plan probe")
	planner := &errorProbePlanner{err: plannerErr}
	verifier := newAcquisitionVerifier(planner)
	workDir := t.TempDir()

	probe, err := verifier.Verify(context.Background(), core.Provider{}, caltrace.Candidate{}, 2, workDir, now)
	if !errors.Is(err, plannerErr) {
		t.Fatalf("Verify() error = %v, want %v", err, plannerErr)
	}
	if probe.Passed || probe.CandidateIndex != 2 || probe.Reason != "probe_plan_failed" || probe.Error == nil || probe.Error.Code != "probe_plan_failed" {
		t.Fatalf("probe = %#v, want failed probe plan result", probe)
	}
	if probe.CreatedAt != now.Format(time.RFC3339Nano) {
		t.Fatalf("probe CreatedAt = %q, want %q", probe.CreatedAt, now.Format(time.RFC3339Nano))
	}
	if planner.workDir != workDir {
		t.Fatalf("planner workDir = %q, want %q", planner.workDir, workDir)
	}
	if _, statErr := os.Stat(planner.workDir); statErr != nil {
		t.Fatalf("probe work dir stat err = %v, want retained dir", statErr)
	}
}

func TestAcquisitionVerifierReportsExecutionFailure(t *testing.T) {
	installAcquisitionTestVerifier(t)
	now := time.Unix(20, 0).UTC()
	verifier := newAcquisitionVerifier(targetProbePlanner{})
	candidate := acquisitionVerifierCandidate()
	provider := core.Provider{
		ID:   "provider_cli",
		Kind: core.ProviderKindCLI,
		Path: writeAcquisitionScript(t, false),
	}

	probe, err := verifier.Verify(context.Background(), provider, candidate, 1, t.TempDir(), now)
	if err == nil {
		t.Fatal("Verify() error = nil, want execution failure")
	}
	if probe.Passed || probe.CandidateIndex != 1 || probe.Reason != "execution_failed" || probe.Error == nil || probe.Error.Code != "execution_failed" {
		t.Fatalf("probe = %#v, want failed execution result", probe)
	}
	if probe.Verifier.ID != "file_exists" {
		t.Fatalf("probe verifier = %#v, want file_exists", probe.Verifier)
	}
	if probe.Inputs["target"] == nil {
		t.Fatalf("probe inputs = %#v, want materialized target", probe.Inputs)
	}
}

func TestAcquisitionVerifierPassesWhenExecutionAndVerifierPass(t *testing.T) {
	installAcquisitionTestVerifier(t)
	now := time.Unix(30, 0).UTC()
	verifier := newAcquisitionVerifier(targetProbePlanner{})
	candidate := acquisitionVerifierCandidate()
	provider := core.Provider{
		ID:   "provider_cli",
		Kind: core.ProviderKindCLI,
		Path: writeAcquisitionScript(t, true),
	}

	probe, err := verifier.Verify(context.Background(), provider, candidate, 0, t.TempDir(), now)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !probe.Passed || probe.CandidateIndex != 0 || probe.Verifier.ID != "file_exists" || len(probe.Evidence) != 1 {
		t.Fatalf("probe = %#v, want passed file_exists probe with evidence", probe)
	}
	if probe.Reason != "file_exists verifier passed" {
		t.Fatalf("probe reason = %q, want verifier pass reason", probe.Reason)
	}
	if probe.Inputs["target"] == nil {
		t.Fatalf("probe inputs = %#v, want materialized target", probe.Inputs)
	}
}

type errorProbePlanner struct {
	err     error
	workDir string
}

func (planner *errorProbePlanner) Plan(_ context.Context, request proposal.ProbePlanRequest) (proposal.ProbePlan, error) {
	planner.workDir = request.WorkDir
	return proposal.ProbePlan{}, planner.err
}

type targetProbePlanner struct{}

func (targetProbePlanner) Plan(_ context.Context, request proposal.ProbePlanRequest) (proposal.ProbePlan, error) {
	return proposal.ProbePlan{
		Inputs:   map[string]any{"target": filepath.Join(request.WorkDir, "output.pdf")},
		Verifier: core.Verifier{ID: "file_exists"},
	}, nil
}

func acquisitionVerifierCandidate() caltrace.Candidate {
	return caltrace.Candidate{
		ProviderID:   "provider_cli",
		CapabilityID: "document.export_pdf",
		Description:  "Export a document to PDF.",
		Execution: core.Execution{
			Kind: core.ExecutionKindCLI,
			Spec: map[string]any{
				"args": []string{"export-pdf", "--target", "{{target}}"},
			},
		},
	}
}

func installAcquisitionTestVerifier(t *testing.T) {
	t.Helper()
	t.Setenv("CAL_HOME", t.TempDir())
	err := runtime.InstallVerifier(runtime.GeneratedVerifierPackage{
		ID: "file_exists",
		VerifyPY: `import json
import os
import sys

request = json.load(sys.stdin)
verifier_id = request["verifier"]["id"]
target = (request.get("inputs") or {}).get("target")
if not isinstance(target, str) or not os.path.exists(target):
    print(json.dumps({"passed": False, "error": {"code": "file_missing", "message": "target file is missing"}}))
    sys.exit(0)
print(json.dumps({
    "passed": True,
    "evidence": [{"id": verifier_id, "type": verifier_id, "content": {"target": target}}],
    "outputs": {"target": target},
}))
`,
	})
	if err != nil {
		t.Fatalf("InstallVerifier(file_exists) error = %v", err)
	}
}
