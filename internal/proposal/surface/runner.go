package surface

import (
	"context"
	"fmt"
	"time"

	"github.com/spacehz-lab/cal/internal/llm"
	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/proposal/policy"
)

// Request asks the surface stage to extract observed CLI surfaces.
type Request struct {
	Provider     *model.Provider
	Observations []model.Observation
	Policy       policy.SurfacePolicy
	MaxItems     int
	Hint         string
}

// Result is the surface stage output.
type Result struct {
	Items   []Item
	Stage   model.ProposalStage
	Attempt model.ProposalAttempt
}

// Item is one candidate surface observed from provider usage text.
type Item struct {
	ID             string                 `json:"id"`
	Kind           string                 `json:"kind,omitempty"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description,omitempty"`
	Usage          string                 `json:"usage,omitempty"`
	EvidenceSource string                 `json:"evidence_source,omitempty"`
	Decision       model.ProposalDecision `json:"decision,omitempty"`
	Reason         string                 `json:"reason,omitempty"`
	Metadata       map[string]any         `json:"metadata,omitempty"`
}

// Runner owns the LLM-backed surface stage.
type Runner struct {
	client llm.Client
}

// NewRunner creates a surface stage runner.
func NewRunner(client llm.Client) *Runner {
	return &Runner{client: client}
}

// Model returns the configured LLM model id.
func (runner *Runner) Model() string {
	if runner == nil || runner.client == nil {
		return ""
	}
	return runner.client.Model()
}

// Run extracts bounded surface items from observations.
func (runner *Runner) Run(ctx context.Context, req *Request) (*Result, error) {
	if runner == nil || runner.client == nil {
		return nil, fmt.Errorf("surface llm client is required")
	}
	started := time.Now()
	response, err := runner.client.Complete(ctx, prompt(req))
	raw := ""
	if response != nil {
		raw = response.Text
	}
	attempt := newAttempt(started, raw, err)
	if err != nil {
		return &Result{Attempt: attempt}, err
	}
	items, stage, err := Parse(raw, req)
	attempt = newAttempt(started, raw, err)
	if err != nil {
		return &Result{Stage: stage, Attempt: attempt}, err
	}
	return &Result{Items: items, Stage: stage, Attempt: attempt}, nil
}
