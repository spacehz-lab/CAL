package control

import (
	caleval "github.com/spacehz-lab/cal/internal/eval"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

// Eval computes aggregate evaluation metrics from local records.
func (svc Service) Eval() (caleval.Metrics, error) {
	return caleval.Compute(svc.store)
}

// GetTrace returns one stored Trace record.
func (svc Service) GetTrace(id string) (caltrace.Trace, bool, error) {
	return svc.store.GetTrace(id)
}
