package proposalflow

import (
	"github.com/spacehz-lab/cal/internal/core"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

type surfaceOutput struct {
	SurfaceItems []surface `json:"surface_items"`
}

type surface struct {
	ID             string                    `json:"id"`
	Kind           string                    `json:"kind,omitempty"`
	Name           string                    `json:"name"`
	Description    string                    `json:"description,omitempty"`
	EvidenceSource string                    `json:"evidence_source,omitempty"`
	Decision       caltrace.ProposalDecision `json:"decision,omitempty"`
	Metadata       map[string]any            `json:"metadata,omitempty"`
}

type capabilitySurface struct {
	ID          string `json:"id"`
	Kind        string `json:"kind,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type existingCapabilityRef struct {
	ID          string `json:"id"`
	Description string `json:"description,omitempty"`
}

type capabilityOutput struct {
	Capabilities []capabilityPlan `json:"capabilities"`
}

type capabilityPlan struct {
	CapabilityID     string   `json:"capability_id"`
	Description      string   `json:"description,omitempty"`
	SourceSurfaceIDs []string `json:"source_surface_ids,omitempty"`
	Confidence       string   `json:"confidence,omitempty"`
}

type bindingOutput struct {
	Candidates     []caltrace.Candidate `json:"candidates"`
	ProbeMaterials []probeMaterial      `json:"probe_material"`
}

type probeMaterial struct {
	CandidateIndex int            `json:"candidate_index"`
	Inputs         map[string]any `json:"inputs,omitempty"`
	Fixtures       []Fixture      `json:"fixtures,omitempty"`
}

type evidenceOutput struct {
	Verify core.VerifySpec `json:"verify"`
}
