package llm

import (
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/proposal"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestPromptBuilderBuildsBoundedProposalPrompt(t *testing.T) {
	prompt := newPromptBuilder().Build(proposal.Request{
		Provider: core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI},
		Observations: []caltrace.Observation{{
			Type:    "help",
			Source:  "tool --help",
			Content: map[string]any{"text": "make-pdf --in <file> --out <pdf>"},
		}},
		ExistingCapabilityIDs: []string{"document.export_pdf"},
	})

	if !strings.Contains(prompt.User, "provider_cli") || !strings.Contains(prompt.User, "existing_capability_ids") || !strings.Contains(prompt.User, "document.export_pdf") {
		t.Fatalf("prompt = %#v, want bounded proposal prompt with provider and existing ids", prompt)
	}
	if strings.Contains(prompt.User, "known_capabilities") || strings.Contains(prompt.User, "available_verifiers") {
		t.Fatalf("prompt user = %q, want no default capability or verifier catalog", prompt.User)
	}
	if !strings.Contains(prompt.System, "proposal JSON") || !strings.Contains(prompt.System, `"verifier_packages"`) || !strings.Contains(prompt.System, `"candidates"`) || !strings.Contains(prompt.System, `"probe_plans"`) || !strings.Contains(prompt.System, `"input_constraints"`) {
		t.Fatalf("system prompt = %q, want proposal schema instructions", prompt.System)
	}
	if !strings.Contains(prompt.System, `"description"`) || !strings.Contains(prompt.System, `"verify_py"`) || !strings.Contains(prompt.System, "standard verifier result JSON") || !strings.Contains(prompt.System, "evidence must always be a JSON array") || !strings.Contains(prompt.System, "provider-independent") {
		t.Fatalf("system prompt = %q, want candidate and verifier contract rules", prompt.System)
	}
	if !strings.Contains(prompt.System, "reusable after Promotion") || !strings.Contains(prompt.System, "fixed literals documented by observations") || !strings.Contains(prompt.System, "Do not require probe-only inputs or fixture-only values") || !strings.Contains(prompt.System, "use Python literals True, False, and None") {
		t.Fatalf("system prompt = %q, want reusable generated verifier rules", prompt.System)
	}
	if !strings.Contains(prompt.System, "validate semantic outcomes") || !strings.Contains(prompt.System, "extract and compare the documented digest value") || !strings.Contains(prompt.System, "labels, filenames, or ASCII whitespace") || !strings.Contains(prompt.System, "source bytes") {
		t.Fatalf("system prompt = %q, want semantic verifier normalization rules", prompt.System)
	}
	if !strings.Contains(prompt.System, "provider observations -> candidate operations") || !strings.Contains(prompt.System, "Do not reverse it") || !strings.Contains(prompt.System, "one verification path is easier to implement") {
		t.Fatalf("system prompt = %q, want one-way inference chain rules", prompt.System)
	}
	if !strings.Contains(prompt.System, "Observations are the only authority") || !strings.Contains(prompt.System, "helper inputs, not proof") || !strings.Contains(prompt.System, "Generate a new capability_id when no existing_capability_ids value matches that full meaning") {
		t.Fatalf("system prompt = %q, want observation-first capability id rules", prompt.System)
	}
	if !strings.Contains(prompt.System, "Generate one verifier_packages[] entry for each probe plan") || !strings.Contains(prompt.System, "verifier_<proposal_local_id>_<hash12>") || !strings.Contains(prompt.System, "inputs.format") {
		t.Fatalf("system prompt = %q, want generated verifier rules", prompt.System)
	}
	if !strings.Contains(prompt.System, "Every placeholder used in execution.spec.args must have a matching probe_plans[].inputs value or fixture input") || !strings.Contains(prompt.System, "if args contain {{filter}}") {
		t.Fatalf("system prompt = %q, want probe input coverage rules", prompt.System)
	}
	if !strings.Contains(prompt.System, "If a probe verifier checks a target artifact") || !strings.Contains(prompt.System, "execution.spec.stdout_path_input") || !strings.Contains(prompt.System, "Do not provide probe_plans[].inputs.target") {
		t.Fatalf("system prompt = %q, want target artifact production rules", prompt.System)
	}
	if !strings.Contains(prompt.System, `"stdout_path_input": "<optional input key, for stdout-producing commands>"`) || !strings.Contains(prompt.System, "Before returning, check each candidate/probe pair") || !strings.Contains(prompt.System, "Commands that print the promised result to stdout must use stdout_path_input") {
		t.Fatalf("system prompt = %q, want explicit stdout artifact contract checklist", prompt.System)
	}
	if strings.Contains(prompt.System, "known_capabilities") || strings.Contains(prompt.System, "available_verifiers") || strings.Contains(prompt.System, "preferred_verifier") {
		t.Fatalf("system prompt = %q, want no default catalog rules", prompt.System)
	}
	if !strings.Contains(prompt.System, "runtime result discriminator") || !strings.Contains(prompt.System, "parameterized capability") || !strings.Contains(prompt.System, "<subject>.<operation>") || !strings.Contains(prompt.System, "narrowest stable object or data type") || !strings.Contains(prompt.System, "not a fixed taxonomy") || !strings.Contains(prompt.System, "subject, operation, and fixed or parameterized result discriminator") || !strings.Contains(prompt.System, "verified documented subset") || !strings.Contains(prompt.System, "Do not rewrite a parameterized candidate into a fixed-value candidate") || !strings.Contains(prompt.System, "include that fixed result discriminator") || !strings.Contains(prompt.System, "lowercase snake_case") {
		t.Fatalf("system prompt = %q, want capability-id granularity rules", prompt.System)
	}
	if strings.Contains(prompt.System, "<domain>.<verb_object>") || strings.Contains(prompt.System, "archive.create_zip") || strings.Contains(prompt.System, "file.hash_sha1") || strings.Contains(prompt.System, "document.convert_format") || strings.Contains(prompt.System, "file.hash_algorithm") {
		t.Fatalf("system prompt = %q, want no capability id examples or fixed taxonomy wording", prompt.System)
	}
	if !strings.Contains(prompt.System, "unrelated operations are not a reason to return an empty candidates array") {
		t.Fatalf("system prompt = %q, want multi-operation provider candidate rules", prompt.System)
	}
	if strings.Contains(prompt.System, "{{workdir}}/output.pdf") {
		t.Fatalf("system prompt = %q, want neutral artifact example", prompt.System)
	}
	if strings.Contains(prompt.System, "file_parse_pdf") || strings.Contains(prompt.System, "file_exists") {
		t.Fatalf("system prompt = %q, want verifier ids selected from prompt payload, not hard-coded prompt rules", prompt.System)
	}
	if strings.Contains(prompt.System, `"metadata"`) || strings.Contains(prompt.System, `"prompt_version"`) || strings.Contains(prompt.System, `"schema_version"`) {
		t.Fatalf("system prompt = %q, want adapter-owned provenance omitted from llm contract", prompt.System)
	}
}
