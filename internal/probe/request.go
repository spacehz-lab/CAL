package probe

import (
	"time"

	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/proposal"
)

const DefaultTimeout = 30 * time.Second

// Request provides one acquisition probe run input.
type Request struct {
	Provider   *model.Provider
	Candidates []model.Candidate
	Plans      []proposal.ProbePlan
	TraceID    string
	WorkRoot   string
	Now        func() time.Time
}

// Result describes probe records produced for candidates.
type Result struct {
	Probes []model.Probe
}

// Target is the internal per-candidate probe unit.
type Target struct {
	CandidateIndex int
	Candidate      *model.Candidate
	Plan           *proposal.ProbePlan
	WorkDir        string
}

// MaterializedPlan is a probe plan with concrete workdir paths.
type MaterializedPlan struct {
	CandidateIndex int
	Inputs         map[string]any
	Verify         model.VerifySpec
	WorkDir        string
}

// Options controls probe runtime bounds and cleanup.
type Options struct {
	Timeout     time.Duration
	KeepWorkdir bool
}

func normalizeOptions(options Options) Options {
	if options.Timeout <= 0 {
		options.Timeout = DefaultTimeout
	}
	return options
}

func (req *Request) now() time.Time {
	if req != nil && req.Now != nil {
		return req.Now().UTC()
	}
	return time.Now().UTC()
}
