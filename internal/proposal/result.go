package proposal

import "github.com/spacehz-lab/cal/internal/model"

// Result is the final proposal output consumed by probe.
type Result struct {
	Candidates  []model.Candidate
	ProbePlans  []ProbePlan
	Diagnostics *model.ProposalTrace
}

// ProbePlan describes how probe should test one candidate.
type ProbePlan struct {
	CandidateIndex int              `json:"candidate_index"`
	Inputs         map[string]any   `json:"inputs,omitempty"`
	Fixtures       []Fixture        `json:"fixtures,omitempty"`
	Verify         model.VerifySpec `json:"verify"`
}

// Fixture describes one file fixture for probe workdir materialization.
type Fixture struct {
	Input    string `json:"input"`
	Filename string `json:"filename"`
	Content  string `json:"content,omitempty"`
}
