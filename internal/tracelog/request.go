package tracelog

import (
	"time"

	"github.com/spacehz-lab/cal/internal/model"
)

// Store is the narrow persistence boundary used by trace logging.
type Store interface {
	SaveTrace(trace *model.Trace) error
}

// Request provides one trace write input.
type Request struct {
	TraceID      string
	StartedAt    string
	Hint         string
	ProviderIDs  []string
	Observations []model.Observation
	Proposal     *model.ProposalTrace
	Candidates   []model.Candidate
	Probes       []model.Probe
	Promotions   []model.Promotion
	Error        *model.RecordError
}

// Result describes one trace write result.
type Result struct {
	Trace model.Trace
}

func nowUTC(now func() time.Time) time.Time {
	if now != nil {
		return now().UTC()
	}
	return time.Now().UTC()
}
