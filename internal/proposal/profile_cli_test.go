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
		Metadata:       map[string]any{"debug": true},
	}})

	if strings.Contains(prompt.User, "decision") || strings.Contains(prompt.User, "evidence_source") || strings.Contains(prompt.User, "metadata") {
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

func TestCLICapabilityPromptConstrainsMerging(t *testing.T) {
	prompt := cliCapabilityPrompt(Request{}, profile{}, DefaultPolicy().Capability, nil)

	for _, want := range []string{
		"compatible Binding inputs, execution shape, and output semantics",
		"Do not merge surfaces that require clearly different inputs",
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

func TestCLIEvidencePromptAllowsSafeWorkdirOutputs(t *testing.T) {
	if strings.Contains(cliEvidenceSystemPrompt, "read-only") {
		t.Fatalf("evidence prompt still says read-only:\n%s", cliEvidenceSystemPrompt)
	}
	for _, want := range []string{
		"read probe fixtures and write declared probe outputs inside the probe workdir",
		`use method="contract", level="L1", checks:[]`,
		"Use contract L0 only when observations are too ambiguous",
		"prefer contains over anchored full-file regex",
		"install, remove, update, upgrade",
	} {
		if !strings.Contains(cliEvidenceSystemPrompt, want) {
			t.Fatalf("evidence prompt missing %q:\n%s", want, cliEvidenceSystemPrompt)
		}
	}
}
