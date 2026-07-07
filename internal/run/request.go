package run

import (
	"context"

	"github.com/spacehz-lab/cal/internal/check"
	"github.com/spacehz-lab/cal/internal/execute"
	"github.com/spacehz-lab/cal/internal/model"
)

// Store describes the persistence operations required by formal runs.
type Store interface {
	GetCapability(id string) (model.Capability, bool, error)
	GetProvider(id string) (model.Provider, bool, error)
	SaveRun(run *model.Run) error
}

// Executor runs selected provider executions.
type Executor interface {
	Run(context.Context, *execute.Request) (*execute.Result, error)
}

// Checker evaluates deterministic verification specs.
type Checker interface {
	Run(context.Context, *check.Request) (*check.Result, error)
}

// Request provides one formal capability run input.
type Request struct {
	CapabilityID   string
	BindingID      string
	ProviderID     string
	Inputs         map[string]any
	Verify         bool
	MinVerifyLevel model.VerifyLevel
}

// Result describes one formal capability run output.
type Result struct {
	Run      *model.Run
	Outputs  execute.Outputs
	Evidence []model.EvidenceRef
}
