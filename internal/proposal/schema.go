package proposal

import (
	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/runtime"
)

// Proposal is the bounded JSON contract for replayed SOP or LLM output.
type Proposal struct {
	Metadata         Metadata                           `json:"metadata,omitempty"`
	VerifierPackages []runtime.GeneratedVerifierPackage `json:"verifier_packages,omitempty"`
	Candidates       []Candidate                        `json:"candidates"`
	ProbePlans       []ProbePlanSpec                    `json:"probe_plans"`
}

// Metadata records proposal provenance without making it trusted evidence.
type Metadata struct {
	Source        string `json:"source,omitempty"`
	PromptVersion string `json:"prompt_version,omitempty"`
	Model         string `json:"model,omitempty"`
	SchemaVersion string `json:"schema_version,omitempty"`
}

// Candidate is a proposed binding candidate.
type Candidate struct {
	ProviderID       string         `json:"provider_id,omitempty"`
	CapabilityID     string         `json:"capability_id"`
	Description      string         `json:"description"`
	Source           string         `json:"source,omitempty"`
	InputConstraints map[string]any `json:"input_constraints,omitempty"`
	Execution        core.Execution `json:"execution"`
}

// ProbePlanSpec is a proposed probe context, not a verification result.
type ProbePlanSpec struct {
	CandidateIndex int            `json:"candidate_index"`
	Inputs         map[string]any `json:"inputs,omitempty"`
	Fixtures       []Fixture      `json:"fixtures,omitempty"`
	Verifier       core.Verifier  `json:"verifier"`
}

// Fixture describes one file that CAL should materialize inside the probe workdir.
type Fixture struct {
	Input    string `json:"input"`
	Filename string `json:"filename"`
	Content  string `json:"content,omitempty"`
}
