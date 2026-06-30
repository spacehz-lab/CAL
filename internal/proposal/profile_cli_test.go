package proposal

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestCLICapabilityPromptUsesSlimSurfaces(t *testing.T) {
	prompt := cliCapabilityPrompt(Request{}, profile{}, DefaultPolicy().Capability, []surface{{
		ID:             "s1",
		Kind:           "command",
		Name:           "dgst",
		Description:    "Calculate message digests.",
		EvidenceSource: "help",
		Decision:       caltrace.ProposalDecisionKeep,
		Reason:         "stable documented operation",
		Metadata:       map[string]any{"debug": true},
	}})

	if strings.Contains(prompt.User, "decision") || strings.Contains(prompt.User, "evidence_source") || strings.Contains(prompt.User, "reason") || strings.Contains(prompt.User, "metadata") {
		t.Fatalf("capability prompt includes Stage1-only fields: %s", prompt.User)
	}

	var payload struct {
		SurfaceItems []capabilitySurface `json:"surface_items"`
	}
	if err := json.Unmarshal([]byte(prompt.User), &payload); err != nil {
		t.Fatalf("decode prompt user: %v", err)
	}
	if len(payload.SurfaceItems) != 1 || payload.SurfaceItems[0].Name != "dgst" || payload.SurfaceItems[0].Description == "" {
		t.Fatalf("surface_items = %#v, want slim surface with description", payload.SurfaceItems)
	}
}

func TestCLISurfacePromptIncludesDecisionRubric(t *testing.T) {
	for _, want := range []string{
		`"reason":"short decision reason"`,
		"Internal decision process:",
		"Does the observed name or description provide a stable operation meaning suitable for Capability planning?",
		"Do not output hidden reasoning steps.",
		"Skip individual algorithms, formats, ciphers, digests",
	} {
		if !strings.Contains(cliSurfaceSystemPrompt, want) {
			t.Fatalf("surface prompt missing %q:\n%s", want, cliSurfaceSystemPrompt)
		}
	}
}

func TestCLICapabilityPromptConstrainsMerging(t *testing.T) {
	prompt := cliCapabilityPrompt(Request{}, profile{}, DefaultPolicy().Capability, nil)

	for _, want := range []string{
		"Internal decision process:",
		"Do not output hidden reasoning steps.",
		"compatible Binding inputs, execution shape, and output semantics",
		"Do not merge surfaces that require clearly different inputs",
		"Do not maintain a format-by-format or algorithm-by-algorithm whitelist",
		`confidence="high" only when the kept surface name and description clearly support`,
		"description must be provider-independent and must describe exactly",
	} {
		if !strings.Contains(prompt.System, want) {
			t.Fatalf("system prompt missing %q:\n%s", want, prompt.System)
		}
	}
}

func TestCLICapabilityPromptIncludesExistingCapabilityDescriptions(t *testing.T) {
	req := Request{
		Catalog: []core.Capability{
			{ID: "document.convert", Description: "Convert documents between formats."},
			{ID: "document.convert.pdf", Description: "Old discriminator id."},
		},
	}
	prompt := cliCapabilityPrompt(req, profile{maxCapabilities: 4}, DefaultPolicy().Capability, nil)

	var payload struct {
		ExistingCapabilities []existingCapabilityRef `json:"existing_capabilities"`
	}
	if err := json.Unmarshal([]byte(prompt.User), &payload); err != nil {
		t.Fatalf("decode prompt user: %v", err)
	}
	if len(payload.ExistingCapabilities) != 1 || payload.ExistingCapabilities[0].ID != "document.convert" || payload.ExistingCapabilities[0].Description == "" {
		t.Fatalf("existing_capabilities = %#v, want valid id with description", payload.ExistingCapabilities)
	}
	if strings.Contains(prompt.User, "existing_capability_ids") {
		t.Fatalf("prompt still contains existing_capability_ids: %s", prompt.User)
	}
}

func TestCLIBindingPromptIncludesControlledProbeRubric(t *testing.T) {
	prompt := cliBindingPrompt(Request{}, profile{}, capabilityPlan{CapabilityID: "text.encode"}, nil)

	for _, want := range []string{
		"Goal:",
		"Stage 3 is Binding.",
		"Do not choose, create, or rename capability_id.",
		"Do not verify, choose verify level, produce checks, claim success, or decide promotion.",
		"Internal decision process:",
		"Keep probe input paths inside {{workdir}} or provide content through fixtures",
		"Do not use shell pipes, redirects, command chaining, or shell-specific syntax",
		"Prefer one most direct, probeable candidate.",
	} {
		if !strings.Contains(prompt.System, want) {
			t.Fatalf("binding prompt missing %q:\n%s", want, prompt.System)
		}
	}
}

func TestCLIEvidencePromptAllowsSafeWorkdirOutputs(t *testing.T) {
	if strings.Contains(cliEvidenceSystemPrompt, "read-only") {
		t.Fatalf("evidence prompt still says read-only:\n%s", cliEvidenceSystemPrompt)
	}
	for _, want := range []string{
		"Stage 4 is Evidence.",
		"Do not claim pass/fail or decide promotion.",
		"Use only subjects, predicates, and params allowed by verify_subject_rules.",
		"Do not output verify.level. CAL derives level locally from method and checks.",
		"Internal decision process:",
		"reads only probe fixtures, and writes only declared probe outputs inside the probe workdir",
		"For contract, checks are advisory and are not executed.",
		"Contract verification does not execute checks and must not include pass/fail claims from execution.",
		"Use checks that prove the result, not the capability name or candidate description.",
		"Do not use fixture-only sample content as durable expected output",
		"prefer contains over anchored full-file regex",
	} {
		if !strings.Contains(cliEvidenceSystemPrompt, want) {
			t.Fatalf("evidence prompt missing %q:\n%s", want, cliEvidenceSystemPrompt)
		}
	}
	if strings.Contains(cliEvidenceSystemPrompt, `"level"`) || strings.Contains(cliEvidenceSystemPrompt, "Choose level") {
		t.Fatalf("evidence prompt still asks the model for verify level:\n%s", cliEvidenceSystemPrompt)
	}
}

func TestCLIEvidencePromptIncludesVerifyParamRules(t *testing.T) {
	prompt := cliEvidencePrompt(Request{Provider: core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI}}, 0, nil, probeMaterial{
		Inputs: map[string]any{"source": "{{workdir}}/input.txt", "target": "{{workdir}}/output.txt"},
	})

	for _, want := range []string{
		`"param_rules"`,
		`"allowed_values":["base64_encode","base64_decode"]`,
		`"allowed_values":["pdf","png","json","text"]`,
		`"allowed_values":["sha1","sha256","sha-1","sha-256","sha_1","sha_256","sha 1","sha 256"]`,
	} {
		if !strings.Contains(prompt.User, want) {
			t.Fatalf("evidence prompt user missing %q:\n%s", want, prompt.User)
		}
	}
}
