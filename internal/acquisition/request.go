package acquisition

import (
	"context"

	"github.com/spacehz-lab/cal/internal/entry"
	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/observe"
	"github.com/spacehz-lab/cal/internal/probe"
	"github.com/spacehz-lab/cal/internal/promote"
	"github.com/spacehz-lab/cal/internal/proposal"
	"github.com/spacehz-lab/cal/internal/tracelog"
)

// Request provides one acquisition run input.
type Request struct {
	ProviderID string
	Hint       string
	TraceID    string
	WorkRoot   string
}

// Result contains the final acquisition trace.
type Result struct {
	Trace model.Trace
}

// ProviderLoader loads one registered provider.
type ProviderLoader interface {
	Load(context.Context, *entry.LoadRequest) (*entry.LoadResult, error)
}

// CatalogStore reads the durable capability catalog.
type CatalogStore interface {
	ListCapabilities() ([]model.Capability, error)
}

// Observer observes one provider.
type Observer interface {
	Observe(context.Context, *observe.Request) (*observe.Result, error)
}

// Proposer proposes candidate bindings and probe plans.
type Proposer interface {
	Run(context.Context, *proposal.Request) (*proposal.Result, error)
}

// Prober verifies proposed candidate bindings.
type Prober interface {
	Run(context.Context, *probe.Request) (*probe.Result, error)
}

// Promoter promotes passed probes into durable bindings.
type Promoter interface {
	Run(context.Context, *promote.Request) (*promote.Result, error)
}

// TraceWriter writes acquisition traces.
type TraceWriter interface {
	Start(context.Context, *tracelog.Request) (*tracelog.Result, error)
	Complete(context.Context, *tracelog.Request) (*tracelog.Result, error)
	Fail(context.Context, *tracelog.Request) (*tracelog.Result, error)
	Cancel(context.Context, *tracelog.Request) (*tracelog.Result, error)
}
