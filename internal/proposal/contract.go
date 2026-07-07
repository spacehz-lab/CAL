package proposal

import (
	"context"

	"github.com/spacehz-lab/cal/internal/proposal/binding"
	"github.com/spacehz-lab/cal/internal/proposal/capability"
	"github.com/spacehz-lab/cal/internal/proposal/evidence"
	"github.com/spacehz-lab/cal/internal/proposal/surface"
)

// SurfaceRunner extracts proposal surface items.
type SurfaceRunner interface {
	Run(context.Context, *surface.Request) (*surface.Result, error)
}

// CapabilityRunner plans provider-independent capabilities.
type CapabilityRunner interface {
	Run(context.Context, *capability.Request) (*capability.Result, error)
}

// BindingRunner plans provider-specific candidate executions.
type BindingRunner interface {
	Run(context.Context, *binding.Request) (*binding.Result, error)
}

// EvidenceRunner plans deterministic verification specs.
type EvidenceRunner interface {
	Run(context.Context, *evidence.Request) (*evidence.Result, error)
}
