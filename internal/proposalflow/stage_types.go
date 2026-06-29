package proposalflow

import (
	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/runtime"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

type surfaceStageOutput struct {
	SurfaceItems []surfaceItem `json:"surface_items"`
}

type surfaceItem struct {
	ID             string                    `json:"id"`
	Kind           string                    `json:"kind,omitempty"`
	Name           string                    `json:"name"`
	Description    string                    `json:"description,omitempty"`
	EvidenceSource string                    `json:"evidence_source,omitempty"`
	Decision       caltrace.ProposalDecision `json:"decision,omitempty"`
	Metadata       map[string]any            `json:"metadata,omitempty"`
}

type capabilitySurfaceItem struct {
	ID          string `json:"id"`
	Kind        string `json:"kind,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type existingCapabilityItem struct {
	ID          string `json:"id"`
	Description string `json:"description,omitempty"`
}

type capabilityStageOutput struct {
	Capabilities []capabilityPlanItem `json:"capabilities"`
}

type capabilityPlanItem struct {
	CapabilityID     string   `json:"capability_id"`
	Description      string   `json:"description,omitempty"`
	SourceSurfaceIDs []string `json:"source_surface_ids,omitempty"`
	Confidence       string   `json:"confidence,omitempty"`
}

type bindingStageOutput struct {
	Candidates     []caltrace.Candidate `json:"candidates"`
	ProbeMaterials []probeMaterial      `json:"probe_material"`
}

type probeMaterial struct {
	CandidateIndex int            `json:"candidate_index"`
	Inputs         map[string]any `json:"inputs,omitempty"`
	Fixtures       []Fixture      `json:"fixtures,omitempty"`
}

type evidenceStageOutput struct {
	VerifierPackages []runtime.GeneratedVerifierPackage `json:"verifier_packages,omitempty"`
	Verifier         core.Verifier                      `json:"verifier"`
}
