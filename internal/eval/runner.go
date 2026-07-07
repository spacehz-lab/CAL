package eval

import (
	"context"
	"fmt"

	"github.com/spacehz-lab/cal/internal/model"
)

// Request filters read-only evaluation metrics.
type Request struct {
	ProviderID   string
	CapabilityID string
}

// Result contains read-only acquisition and reuse metrics.
type Result struct {
	Acquisition AcquisitionMetrics `json:"acquisition"`
	Reuse       ReuseMetrics       `json:"reuse"`
	Capability  CapabilityMetrics  `json:"capability"`
}

// Store reads durable records needed by eval.
type Store interface {
	ListTraces() ([]model.Trace, error)
	ListRuns() ([]model.Run, error)
	ListCapabilities() ([]model.Capability, error)
}

// Runner owns read-only metric aggregation.
type Runner struct {
	store Store
}

// NewRunner creates an eval runner.
func NewRunner(store Store) *Runner {
	return &Runner{store: store}
}

// Run reads durable records and returns summary metrics.
func (runner *Runner) Run(ctx context.Context, req *Request) (*Result, error) {
	if runner == nil || runner.store == nil {
		return nil, fmt.Errorf("eval store is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	filter := requestFilter(req)
	records, err := runner.loadRecords(ctx, filter)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return &Result{
		Acquisition: aggregateAcquisition(records.Traces),
		Reuse:       aggregateReuse(records.Runs),
		Capability:  aggregateCapability(records.Capabilities),
	}, nil
}

func requestFilter(req *Request) recordFilter {
	if req == nil {
		return recordFilter{}
	}
	return recordFilter{ProviderID: req.ProviderID, CapabilityID: req.CapabilityID}
}
