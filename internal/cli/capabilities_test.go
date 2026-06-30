package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/store"
)

func TestCapabilityListReturnsPromotedSummary(t *testing.T) {
	home := t.TempDir()
	startCLITestCald(t, home)
	s, err := store.Open(home)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := s.PutCapability(core.Capability{
		ID:          "document.convert",
		Description: "Export a document to PDF.",
		Bindings: []core.Binding{
			{
				ID:           "binding_promoted",
				CapabilityID: "document.convert",
				ProviderID:   "provider_fake",
				Execution:    core.Execution{Kind: core.ExecutionKindCLI, Spec: map[string]any{"secret": "do not expose"}},
				Verify: &core.VerifySpec{
					Level:  core.VerifyLevelL2,
					Method: core.VerifyMethodExecute,
					Checks: []core.VerifyCheck{{Subject: core.VerifySubject{Type: core.VerifySubjectFile, Input: "target"}, Predicate: core.VerifyPredicateExists}},
				},
				Evidence: []core.EvidenceRef{{ID: "evidence_fake"}},
				State:    core.BindingStatePromoted,
			},
		},
	}); err != nil {
		t.Fatalf("PutCapability() error = %v", err)
	}

	var out bytes.Buffer
	cmd := NewRootCommand(Config{Home: home, Out: &out, Err: io.Discard})
	cmd.SetArgs([]string{"capabilities", "list", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var response struct {
		Count        int `json:"count"`
		Capabilities []struct {
			ID       string `json:"id"`
			Bindings struct {
				Available    int      `json:"available"`
				ProviderIDs  []string `json:"provider_ids"`
				VerifyLevels []string `json:"verify_levels"`
			} `json:"bindings"`
			Execution any `json:"execution"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v\n%s", err, out.String())
	}
	if response.Count != 1 || len(response.Capabilities) != 1 {
		t.Fatalf("response = %#v, want one promoted capability", response)
	}
	capability := response.Capabilities[0]
	if capability.ID != "document.convert" || capability.Bindings.Available != 1 {
		t.Fatalf("capability = %#v, want promoted summary only", capability)
	}
	if capability.Execution != nil {
		t.Fatalf("capability list exposed execution: %#v", capability.Execution)
	}
	if len(capability.Bindings.ProviderIDs) != 1 || capability.Bindings.ProviderIDs[0] != "provider_fake" {
		t.Fatalf("provider_ids = %#v, want provider_fake", capability.Bindings.ProviderIDs)
	}
	if len(capability.Bindings.VerifyLevels) != 1 || capability.Bindings.VerifyLevels[0] != "L2" {
		t.Fatalf("verify_levels = %#v, want L2", capability.Bindings.VerifyLevels)
	}
}

func TestCapabilityListTextEmpty(t *testing.T) {
	home := t.TempDir()
	startCLITestCald(t, home)
	output, err := executeRoot(home, "capabilities", "list")
	if err != nil {
		t.Fatalf("capability list error = %v\n%s", err, output)
	}
	if !strings.Contains(output, "no capabilities") {
		t.Fatalf("capability list output = %q, want empty text", output)
	}
}
