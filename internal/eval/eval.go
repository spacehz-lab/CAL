package eval

import (
	"log/slog"
	"time"

	"github.com/spacehz-lab/cal/internal/core"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

// Store is the read surface needed to compute metrics.
type Store interface {
	ListProviders() ([]core.Provider, error)
	ListCapabilities() ([]core.Capability, error)
	ListRuns() ([]core.Run, error)
	ListTraces() ([]caltrace.Trace, error)
}

// Evaluator reads local records and aggregates experiment metrics.
type Evaluator struct {
	store Store
}

// NewEvaluator builds an evaluator for a read-only metrics store.
func NewEvaluator(s Store) Evaluator {
	return Evaluator{store: s}
}

// Compute aggregates metrics from local CAL records.
func Compute(s Store) (Metrics, error) {
	return NewEvaluator(s).Compute()
}

// Compute aggregates metrics from local CAL records.
func (e Evaluator) Compute() (Metrics, error) {
	started := time.Now()
	records, err := e.load()
	if err != nil {
		slog.Warn("eval compute failed",
			"stage", "load",
			"duration_ms", time.Since(started).Milliseconds(),
		)
		return Metrics{}, err
	}
	metrics := Metrics{
		Summary:     records.summary(),
		Acquisition: records.acquisition(),
		Reuse:       records.reuse(),
	}
	slog.Info("eval compute completed",
		"providers", metrics.Summary.Providers,
		"capabilities", metrics.Summary.Capabilities,
		"bindings", metrics.Summary.Bindings,
		"promoted_bindings", metrics.Summary.PromotedBindings,
		"traces", metrics.Summary.Traces,
		"runs", metrics.Summary.Runs,
		"duration_ms", time.Since(started).Milliseconds(),
	)
	return metrics, nil
}

func (e Evaluator) load() (records, error) {
	providers, err := e.store.ListProviders()
	if err != nil {
		return records{}, err
	}
	capabilities, err := e.store.ListCapabilities()
	if err != nil {
		return records{}, err
	}
	runs, err := e.store.ListRuns()
	if err != nil {
		return records{}, err
	}
	traces, err := e.store.ListTraces()
	if err != nil {
		return records{}, err
	}
	return records{
		providers:    providers,
		capabilities: capabilities,
		runs:         runs,
		traces:       traces,
	}, nil
}
