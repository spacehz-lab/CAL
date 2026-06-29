package discovery

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/proposalflow"
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
	if probe.Verify.Level != core.VerifyLevelL2 {
		t.Fatalf("probe verify = %#v, want L2", probe.Verify)
	}
	if probe.Inputs["target"] != filepath.Join(workDir, "output.pdf") {
		t.Fatalf("probe inputs = %#v, want materialized target", probe.Inputs)
	}
}

func TestAcquisitionVerifierReportsExecutionTimeout(t *testing.T) {
	now := time.Unix(25, 0).UTC()
	probe, err := verifyProbe(context.Background(), probeVerification{
		Provider:       core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI, Path: writeSlowAcquisitionScript(t)},
		Candidate:      acquisitionVerifierCandidate(),
		Plan:           targetProbePlan(),
		CandidateIndex: 1,
		WorkDir:        t.TempDir(),
		Now:            now,
		Timeout:        10 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("Verify() error = nil, want execution timeout")
	}
	if probe.Passed || probe.Reason != "execution_timeout" || probe.Error == nil || probe.Error.Code != "execution_timeout" {
		t.Fatalf("probe = %#v, want execution_timeout probe", probe)
	}
}

func TestAcquisitionVerifierPassesWhenExecutionAndVerifyPass(t *testing.T) {
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
	if !probe.Passed || probe.CandidateIndex != 0 || probe.Verify.Level != core.VerifyLevelL2 || len(probe.Evidence) != 1 {
		t.Fatalf("probe = %#v, want passed L2 probe with evidence", probe)
	}
	if probe.Reason != "L2 verify checks passed" {
		t.Fatalf("probe reason = %q, want verify pass reason", probe.Reason)
	}
	if probe.Inputs["target"] != filepath.Join(workDir, "output.pdf") {
		t.Fatalf("probe inputs = %#v, want materialized target", probe.Inputs)
	}
}

func TestAcquisitionVerifierAcceptsContractWithoutExecution(t *testing.T) {
	now := time.Unix(40, 0).UTC()
	probe, err := verifyProbe(context.Background(), probeVerification{
		Provider:       core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI, Path: "/missing/provider"},
		Candidate:      acquisitionVerifierCandidate(),
		Plan:           contractProbePlan(),
		CandidateIndex: 0,
		WorkDir:        t.TempDir(),
		Now:            now,
	})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !probe.Passed || probe.Verify.Level != core.VerifyLevelL1 || probe.Verify.Method != core.VerifyMethodContract || len(probe.Evidence) != 1 {
		t.Fatalf("probe = %#v, want passed contract L1 probe with evidence", probe)
	}
	if probe.Evidence[0].ID != contractEvidenceID {
		t.Fatalf("evidence = %#v, want contract evidence", probe.Evidence)
	}
}

func TestAcquisitionVerifierRejectsInvalidVerifyPlan(t *testing.T) {
	now := time.Unix(45, 0).UTC()
	probe, err := verifyProbe(context.Background(), probeVerification{
		Provider:       core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI, Path: "/missing/provider"},
		Candidate:      acquisitionVerifierCandidate(),
		Plan:           invalidContractProbePlan(),
		CandidateIndex: 0,
		WorkDir:        t.TempDir(),
		Now:            now,
	})
	if err == nil {
		t.Fatal("Verify() error = nil, want invalid verify plan error")
	}
	if probe.Passed || probe.Reason != "verification_plan_invalid" {
		t.Fatalf("probe = %#v, want invalid verify plan probe", probe)
	}
}

func TestAcquisitionVerifierRejectsL0WithoutExecution(t *testing.T) {
	now := time.Unix(50, 0).UTC()
	probe, err := verifyProbe(context.Background(), probeVerification{
		Provider:       core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI, Path: "/missing/provider"},
		Candidate:      acquisitionVerifierCandidate(),
		Plan:           l0ProbePlan(),
		CandidateIndex: 0,
		WorkDir:        t.TempDir(),
		Now:            now,
	})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if probe.Passed || probe.Verify.Level != core.VerifyLevelL0 || probe.Reason != l0Reason {
		t.Fatalf("probe = %#v, want non-passed L0 probe", probe)
	}
}

func targetProbePlan() proposalflow.ProbePlan {
	return proposalflow.ProbePlan{
		Inputs: map[string]any{"target": "{{workdir}}/output.pdf"},
		Verify: fileExistsVerifySpec(),
	}
}

func contractProbePlan() proposalflow.ProbePlan {
	return proposalflow.ProbePlan{
		Inputs: map[string]any{"target": "{{workdir}}/output.pdf"},
		Verify: core.VerifySpec{Level: core.VerifyLevelL1, Method: core.VerifyMethodContract},
	}
}

func invalidContractProbePlan() proposalflow.ProbePlan {
	return proposalflow.ProbePlan{
		Inputs: map[string]any{"target": "{{workdir}}/output.pdf"},
		Verify: core.VerifySpec{Level: core.VerifyLevelL2, Method: core.VerifyMethodContract},
	}
}

func l0ProbePlan() proposalflow.ProbePlan {
	return proposalflow.ProbePlan{
		Inputs: map[string]any{"target": "{{workdir}}/output.pdf"},
		Verify: core.VerifySpec{Level: core.VerifyLevelL0, Method: core.VerifyMethodContract},
	}
}

func writeSlowAcquisitionScript(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "slow-cli")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nsleep 1\n"), 0o755); err != nil {
		t.Fatalf("write slow acquisition script: %v", err)
	}
	return path
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
