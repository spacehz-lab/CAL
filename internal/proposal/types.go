package proposal

import (
	"context"

	"github.com/spacehz-lab/cal/internal/core"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

// Proposer produces a selected, executable Proposal result for one provider.
type Proposer interface {
	Propose(context.Context, Request) (Result, error)
}

// Request is the input to one provider Proposal run.
type Request struct {
	Provider     core.Provider
	Observations []caltrace.Observation
	Catalog      []core.Capability
	DebugFilter  string
	TraceID      string
}

// Result is the selected, executable Proposal output consumed by discovery.
// Candidates and ProbePlans must be one-to-one after filtering, de-duplication, limiting, and re-indexing.
type Result struct {
	Candidates  []caltrace.Candidate
	ProbePlans  []ProbePlan
	Diagnostics *caltrace.ProposalTrace
}

// ProbePlan is the proposed probe context for one candidate.
type ProbePlan struct {
	CandidateIndex int             `json:"candidate_index"`
	Inputs         map[string]any  `json:"inputs,omitempty"`
	Fixtures       []Fixture       `json:"fixtures,omitempty"`
	Verify         core.VerifySpec `json:"verify"`
}

// Fixture describes one file to materialize inside the probe workdir.
type Fixture struct {
	Input    string `json:"input"`
	Filename string `json:"filename"`
	Content  string `json:"content,omitempty"`
}
