package discovery

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/proposalflow"
	"github.com/spacehz-lab/cal/internal/runtime"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestAcquisitionVerifierRequiresWorkDir(t *testing.T) {
	now := time.Unix(10, 0).UTC()

	_, err := verifyProbe(context.Background(), probeVerification{
		CandidateIndex: 2,
		Now:            now,
	})
	if err == nil {
		t.Fatal("Verify() error = nil, want work dir error")
	}
}

func TestAcquisitionVerifierReportsExecutionFailure(t *testing.T) {
	installAcquisitionTestVerifier(t)
	now := time.Unix(20, 0).UTC()
	candidate := acquisitionVerifierCandidate()
	provider := core.Provider{
		ID:   "provider_cli",
		Kind: core.ProviderKindCLI,
		Path: writeAcquisitionScript(t, false),
	}
	workDir := t.TempDir()

	probe, err := verifyProbe(context.Background(), probeVerification{
		Provider:       provider,
		Candidate:      candidate,
		Plan:           targetProbePlan(),
		CandidateIndex: 1,
		WorkDir:        workDir,
		Now:            now,
	})
	if err == nil {
		t.Fatal("Verify() error = nil, want execution failure")
	}
	if probe.Passed || probe.CandidateIndex != 1 || probe.Reason != "execution_failed" || probe.Error == nil || probe.Error.Code != "execution_failed" {
		t.Fatalf("probe = %#v, want failed execution result", probe)
	}
	if probe.Verifier.ID != "file_exists" {
		t.Fatalf("probe verifier = %#v, want file_exists", probe.Verifier)
	}
	if probe.Inputs["target"] != filepath.Join(workDir, "output.pdf") {
		t.Fatalf("probe inputs = %#v, want materialized target", probe.Inputs)
	}
}

func TestAcquisitionVerifierPassesWhenExecutionAndVerifierPass(t *testing.T) {
	installAcquisitionTestVerifier(t)
	now := time.Unix(30, 0).UTC()
	candidate := acquisitionVerifierCandidate()
	provider := core.Provider{
		ID:   "provider_cli",
		Kind: core.ProviderKindCLI,
		Path: writeAcquisitionScript(t, true),
	}
	workDir := t.TempDir()

	probe, err := verifyProbe(context.Background(), probeVerification{
		Provider:       provider,
		Candidate:      candidate,
		Plan:           targetProbePlan(),
		CandidateIndex: 0,
		WorkDir:        workDir,
		Now:            now,
	})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !probe.Passed || probe.CandidateIndex != 0 || probe.Verifier.ID != "file_exists" || len(probe.Evidence) != 1 {
		t.Fatalf("probe = %#v, want passed file_exists probe with evidence", probe)
	}
	if probe.Reason != "file_exists verifier passed" {
		t.Fatalf("probe reason = %q, want verifier pass reason", probe.Reason)
	}
	if probe.Inputs["target"] != filepath.Join(workDir, "output.pdf") {
		t.Fatalf("probe inputs = %#v, want materialized target", probe.Inputs)
	}
}

func targetProbePlan() proposalflow.ProbePlan {
	return proposalflow.ProbePlan{
		Inputs:   map[string]any{"target": "{{workdir}}/output.pdf"},
		Verifier: core.Verifier{ID: "file_exists"},
	}
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
