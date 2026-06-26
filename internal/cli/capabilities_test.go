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
		ID:          "document.export_pdf",
		Description: "Export a document to PDF.",
		Bindings: []core.Binding{
			{
				ID:           "binding_promoted",
				CapabilityID: "document.export_pdf",
				ProviderID:   "provider_fake",
				Execution:    core.Execution{Kind: core.ExecutionKindCLI, Spec: map[string]any{"secret": "do not expose"}},
				Verifier:     &core.Verifier{ID: "file_exists"},
				Evidence:     []core.EvidenceRef{{ID: "evidence_fake"}},
				State:        core.BindingStatePromoted,
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
				Available   int      `json:"available"`
				ProviderIDs []string `json:"provider_ids"`
				Verifiers   []string `json:"verifiers"`
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
	if capability.ID != "document.export_pdf" || capability.Bindings.Available != 1 {
		t.Fatalf("capability = %#v, want promoted summary only", capability)
	}
	if capability.Execution != nil {
		t.Fatalf("capability list exposed execution: %#v", capability.Execution)
	}
	if len(capability.Bindings.ProviderIDs) != 1 || capability.Bindings.ProviderIDs[0] != "provider_fake" {
		t.Fatalf("provider_ids = %#v, want provider_fake", capability.Bindings.ProviderIDs)
	}
	if len(capability.Bindings.Verifiers) != 1 || capability.Bindings.Verifiers[0] != "file_exists" {
		t.Fatalf("verifiers = %#v, want file_exists", capability.Bindings.Verifiers)
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
