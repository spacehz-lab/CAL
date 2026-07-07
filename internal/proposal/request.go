package proposal

import "github.com/spacehz-lab/cal/internal/model"

// Request is one provider proposal run input.
type Request struct {
	Provider     *model.Provider
	Observations []model.Observation
	Catalog      []model.Capability
	Hint         string
	TraceID      string
}
