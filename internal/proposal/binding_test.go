package proposal

import (
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestNormalizeBindingStageKeepsValidCandidate(t *testing.T) {
	output, stage := normalizeBindings(bindingRequest(), bindingProfile(2), bindingCapability(), bindingOutput{
		Candidates: []caltrace.Candidate{bindingCandidate([]string{"convert", "{{source}}", "{{target}}"})},
		ProbeMaterials: []probeMaterial{{
			CandidateIndex: 0,
			Inputs:         map[string]any{"target": "{{workdir}}/out.txt"},
			Fixtures:       []Fixture{{Input: "source", Filename: "input.txt", Content: "hello"}},
		}},
	})

	if len(output.Candidates) != 1 || len(output.ProbeMaterials) != 1 {
		t.Fatalf("output = %#v, want one selected candidate and probe material", output)
	}
	if stage.Name != caltrace.ProposalStageBinding || stage.Summary[caltrace.ProposalSummaryKeep] != 1 || stage.Summary[caltrace.ProposalSummarySelected] != 1 {
		t.Fatalf("stage = %#v, want one kept binding", stage)
	}
}

func TestNormalizeBindingStageSkipsMissingProbeInput(t *testing.T) {
	output, stage := normalizeBindings(bindingRequest(), bindingProfile(2), bindingCapability(), bindingOutput{
		Candidates: []caltrace.Candidate{bindingCandidate([]string{"convert", "{{source}}", "{{target}}"})},
		ProbeMaterials: []probeMaterial{{
			CandidateIndex: 0,
			Inputs:         map[string]any{"target": "{{workdir}}/out.txt"},
		}},
	})

	if len(output.Candidates) != 0 {
		t.Fatalf("output = %#v, want missing source input skipped", output)
	}
	if stage.Summary[caltrace.ProposalSummarySkip] != 1 || stage.Summary[caltrace.ProposalSummarySelected] != 0 {
		t.Fatalf("stage = %#v, want skipped binding", stage)
	}
	if got := bindingStageItemReason(t, stage, 0); got != "missing_probe_input:source" {
		t.Fatalf("reason = %q, want missing source input", got)
	}
}

func TestNormalizeBindingStageSkipsMissingProbeMaterial(t *testing.T) {
	output, stage := normalizeBindings(bindingRequest(), bindingProfile(2), bindingCapability(), bindingOutput{
		Candidates: []caltrace.Candidate{bindingCandidate([]string{"convert", "{{source}}"})},
	})

	if len(output.Candidates) != 0 {
		t.Fatalf("output = %#v, want missing probe material skipped", output)
	}
	if stage.Summary[caltrace.ProposalSummarySkip] != 1 {
		t.Fatalf("stage = %#v, want skipped binding", stage)
	}
	if got := bindingStageItemReason(t, stage, 0); got != bindingReasonMissingProbeMaterial {
		t.Fatalf("reason = %q, want missing probe material", got)
	}
}

func TestNormalizeBindingStageSkipsDuplicateProbeMaterialIndex(t *testing.T) {
	output, stage := normalizeBindings(bindingRequest(), bindingProfile(2), bindingCapability(), bindingOutput{
		Candidates: []caltrace.Candidate{bindingCandidate([]string{"convert", "{{source}}"})},
		ProbeMaterials: []probeMaterial{
			{CandidateIndex: 0, Fixtures: []Fixture{{Input: "source", Filename: "a.txt"}}},
			{CandidateIndex: 0, Fixtures: []Fixture{{Input: "source", Filename: "b.txt"}}},
		},
	})

	if len(output.Candidates) != 0 {
		t.Fatalf("output = %#v, want duplicate probe material skipped", output)
	}
	if stage.Summary[caltrace.ProposalSummarySkip] != 1 {
		t.Fatalf("stage = %#v, want skipped binding", stage)
	}
	if got := bindingStageItemReason(t, stage, 0); got != bindingReasonDuplicateProbeMaterial {
		t.Fatalf("reason = %q, want duplicate probe material", got)
	}
}

func TestNormalizeBindingStageSkipsUnknownInputConstraint(t *testing.T) {
	candidate := bindingCandidate([]string{"convert", "{{source}}"})
	candidate.InputConstraints = map[string]any{"format": map[string]any{"type": "string"}}
	output, stage := normalizeBindings(bindingRequest(), bindingProfile(2), bindingCapability(), bindingOutput{
		Candidates:     []caltrace.Candidate{candidate},
		ProbeMaterials: []probeMaterial{{CandidateIndex: 0, Fixtures: []Fixture{{Input: "source", Filename: "input.txt"}}}},
	})

	if len(output.Candidates) != 0 {
		t.Fatalf("output = %#v, want unknown input constraint skipped", output)
	}
	if stage.Summary[caltrace.ProposalSummarySkip] != 1 {
		t.Fatalf("stage = %#v, want skipped binding", stage)
	}
	if got := bindingStageItemReason(t, stage, 0); got != "unknown_input_constraint:format" {
		t.Fatalf("reason = %q, want unknown input constraint", got)
	}
}

func TestNormalizeBindingStageSkipsInvalidInputConstraint(t *testing.T) {
	candidate := bindingCandidate([]string{"convert", "{{source}}"})
	candidate.InputConstraints = map[string]any{"source": "path"}
	output, stage := normalizeBindings(bindingRequest(), bindingProfile(2), bindingCapability(), bindingOutput{
		Candidates:     []caltrace.Candidate{candidate},
		ProbeMaterials: []probeMaterial{{CandidateIndex: 0, Fixtures: []Fixture{{Input: "source", Filename: "input.txt"}}}},
	})

	if len(output.Candidates) != 0 {
		t.Fatalf("output = %#v, want invalid input constraint skipped", output)
	}
	if stage.Summary[caltrace.ProposalSummarySkip] != 1 {
		t.Fatalf("stage = %#v, want skipped binding", stage)
	}
	if got := bindingStageItemReason(t, stage, 0); got != "invalid_input_constraint:source" {
		t.Fatalf("reason = %q, want invalid input constraint", got)
	}
}

func TestNormalizeBindingStageSkipsProviderExecutableInArgs(t *testing.T) {
	output, stage := normalizeBindings(bindingRequest(), bindingProfile(2), bindingCapability(), bindingOutput{
		Candidates:     []caltrace.Candidate{bindingCandidate([]string{"brew", "install", "{{package}}"})},
		ProbeMaterials: []probeMaterial{{CandidateIndex: 0, Inputs: map[string]any{"package": "hello"}}},
	})

	if len(output.Candidates) != 0 {
		t.Fatalf("output = %#v, want provider executable skipped", output)
	}
	if stage.Summary[caltrace.ProposalSummarySkip] != 1 {
		t.Fatalf("stage = %#v, want skipped binding", stage)
	}
	if got := bindingStageItemReason(t, stage, 0); got != bindingReasonProviderExecutableInArgs {
		t.Fatalf("reason = %q, want provider executable in args", got)
	}
}

func TestNormalizeBindingStageSkipsStringCLIArgs(t *testing.T) {
	candidate := bindingCandidate(nil)
	candidate.Execution.Spec[core.ExecutionSpecArgs] = "system doctor --json --output {{target}}"
	output, stage := normalizeBindings(bindingRequest(), bindingProfile(2), bindingCapability(), bindingOutput{
		Candidates:     []caltrace.Candidate{candidate},
		ProbeMaterials: []probeMaterial{{CandidateIndex: 0, Inputs: map[string]any{"target": "{{workdir}}/report.json"}}},
	})

	if len(output.Candidates) != 0 {
		t.Fatalf("output = %#v, want string args skipped", output)
	}
	if got := bindingStageItemReason(t, stage, 0); got != "invalid_cli_args_type:string" {
		t.Fatalf("reason = %q, want invalid string args type", got)
	}
}

func TestNormalizeBindingStageAcceptsStdoutPathInput(t *testing.T) {
	candidate := bindingCandidate([]string{"sha1", "{{source}}"})
	candidate.Execution.Spec[core.ExecutionSpecStdoutPathInput] = "target"
	output, stage := normalizeBindings(bindingRequest(), bindingProfile(2), bindingCapability(), bindingOutput{
		Candidates: []caltrace.Candidate{candidate},
		ProbeMaterials: []probeMaterial{{
			CandidateIndex: 0,
			Inputs:         map[string]any{"target": "{{workdir}}/out.txt"},
			Fixtures:       []Fixture{{Input: "source", Filename: "input.txt"}},
		}},
	})

	if len(output.Candidates) != 1 {
		t.Fatalf("output = %#v, want stdout path candidate selected", output)
	}
	if stage.Summary[caltrace.ProposalSummaryKeep] != 1 {
		t.Fatalf("stage = %#v, want kept binding", stage)
	}
}

func TestNormalizeBindingStageLimitsCandidates(t *testing.T) {
	output, stage := normalizeBindings(bindingRequest(), bindingProfile(2), bindingCapability(), bindingOutput{
		Candidates: []caltrace.Candidate{
			bindingCandidate([]string{"a", "{{source}}"}),
			bindingCandidate([]string{"b", "{{source}}"}),
			bindingCandidate([]string{"c", "{{source}}"}),
		},
		ProbeMaterials: []probeMaterial{
			{CandidateIndex: 0, Fixtures: []Fixture{{Input: "source", Filename: "a.txt"}}},
			{CandidateIndex: 1, Fixtures: []Fixture{{Input: "source", Filename: "b.txt"}}},
			{CandidateIndex: 2, Fixtures: []Fixture{{Input: "source", Filename: "c.txt"}}},
		},
	})

	if len(output.Candidates) != 2 {
		t.Fatalf("output = %#v, want two selected candidates", output)
	}
	if stage.Summary[caltrace.ProposalSummaryKeep] != 2 || stage.Summary[caltrace.ProposalSummaryDefer] != 1 || stage.Summary[caltrace.ProposalSummarySelected] != 2 {
		t.Fatalf("stage = %#v, want two kept and one deferred", stage)
	}
	if got := bindingStageItemReason(t, stage, 2); got != bindingReasonCandidateLimit {
		t.Fatalf("reason = %q, want candidate limit", got)
	}
}

func bindingStageItemReason(t *testing.T, stage caltrace.ProposalStage, index int) string {
	t.Helper()
	if len(stage.Items) <= index {
		t.Fatalf("stage items = %#v, want index %d", stage.Items, index)
	}
	return stage.Items[index].Reason
}

func bindingRequest() Request {
	return Request{Provider: core.Provider{ID: "provider_brew", Kind: core.ProviderKindCLI, Path: "/opt/homebrew/bin/brew"}}
}

func bindingProfile(maxCandidates int) profile {
	return profile{maxCandidatesPerCapability: maxCandidates}
}

func bindingCapability() capabilityPlan {
	return capabilityPlan{CapabilityID: "package.install", Description: "Install a package."}
}

func bindingCandidate(args []string) caltrace.Candidate {
	return caltrace.Candidate{
		Description: "Install a package.",
		Execution: core.Execution{
			Kind: core.ExecutionKindCLI,
			Spec: map[string]any{core.ExecutionSpecArgs: args},
		},
	}
}
