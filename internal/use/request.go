package use

import (
	"context"

	"github.com/spacehz-lab/cal/internal/model"
	runpkg "github.com/spacehz-lab/cal/internal/run"
)

const (
	CodeInvalidUseInput     = "invalid_use_input"
	CodeNoMatch             = "no_match"
	CodeMissingInputs       = "missing_inputs"
	CodeAmbiguous           = "ambiguous"
	CodeLLMSelectionFailed  = "llm_selection_failed"
	CodeInvalidLLMSelection = "invalid_llm_selection"
	CodeArtifactPathFailed  = "artifact_path_failed"
	CodeRunFailed           = "run_failed"
	CodeUseStoreFailed      = "use_store_failed"
)

// Store describes the catalog operation required by use.
type Store interface {
	ListCapabilities() ([]model.Capability, error)
}

// Executor executes a planned formal run.
type Executor interface {
	Run(context.Context, *runpkg.Request) (*runpkg.Result, error)
}

// Request provides one intent-level reuse input.
type Request struct {
	Intent         string
	Inputs         map[string]any
	ProviderID     string
	Verify         bool
	MinVerifyLevel model.VerifyLevel
}

// Result describes one intent-level reuse result.
type Result struct {
	ID         string
	Intent     string
	Selection  *Selection
	Run        *model.Run
	Status     model.RunStatus
	StartedAt  string
	FinishedAt string
	DurationMS int64
	Error      *model.RecordError
}

// Selection describes the promoted binding selected for one use request.
type Selection struct {
	Source               string
	CapabilityID         string
	BindingID            string
	ProviderID           string
	Reason               string
	CandidatesConsidered int
}
