package promote

import (
	"time"

	"github.com/spacehz-lab/cal/internal/model"
)

// Store is the narrow persistence boundary used by promotion.
type Store interface {
	GetCapability(id string) (model.Capability, bool, error)
	SaveCapability(capability *model.Capability) error
}

// Request provides one promotion run input.
type Request struct {
	Candidates []model.Candidate
	Probes     []model.Probe
}

// Result describes promotions created during one run.
type Result struct {
	Promotions []model.Promotion
}

// Target is the internal per-passed-probe promotion unit.
type Target struct {
	CandidateIndex int
	Candidate      *model.Candidate
	Probe          *model.Probe
}

func nowUTC(now func() time.Time) time.Time {
	if now != nil {
		return now().UTC()
	}
	return time.Now().UTC()
}
