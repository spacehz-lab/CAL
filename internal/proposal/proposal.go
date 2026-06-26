package proposal

import (
	"context"

	"github.com/spacehz-lab/cal/internal/core"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

// Proposer creates candidate proposals from observations.
type Proposer interface {
	Propose(context.Context, Request) (Response, error)
}

// Request is the input to candidate proposal generation.
type Request struct {
	Provider              core.Provider
	Observations          []caltrace.Observation
	ExistingCapabilityIDs []string
	Hint                  string
}

// Response is the candidate proposal result.
type Response struct {
	Candidates []caltrace.Candidate
}

// ProbePlanRequest contains the bounded context needed to materialize one probe.
type ProbePlanRequest struct {
	Candidate caltrace.Candidate
	WorkDir   string
}

// ProbePlan describes the inputs and deterministic verifier for one candidate probe.
type ProbePlan struct {
	Inputs   map[string]any
	Verifier core.Verifier
}

// ProbePlanner materializes probe inputs and a verifier without deciding pass/fail.
type ProbePlanner interface {
	Plan(context.Context, ProbePlanRequest) (ProbePlan, error)
}
