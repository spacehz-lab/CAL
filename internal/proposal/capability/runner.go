package capability

import (
	"context"
	"fmt"
	"time"

	"github.com/spacehz-lab/cal/internal/llm"
	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/proposal/policy"
)

// Request asks the capability stage to plan reusable capabilities.
type Request struct {
	Provider *model.Provider
	Surfaces []SurfaceItem
	Catalog  []model.Capability
	Policy   policy.CapabilityPolicy
	Hint     string
	MaxPlans int
}

// SurfaceItem is the capability stage's view of a surface item.
type SurfaceItem struct {
	ID          string
	Kind        string
	Name        string
	Description string
}

// Plan is one provider-independent capability plan.
type Plan struct {
	CapabilityID     string   `json:"capability_id"`
	Description      string   `json:"description,omitempty"`
	SourceSurfaceIDs []string `json:"source_surface_ids,omitempty"`
	Confidence       string   `json:"confidence,omitempty"`
}

// Result is the capability stage output.
type Result struct {
	Plans   []Plan
	Stage   model.ProposalStage
	Attempt model.ProposalAttempt
}

// Runner owns the LLM-backed capability stage.
type Runner struct {
	client llm.Client
}

// NewRunner creates a capability stage runner.
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

// Run plans provider-independent capabilities.
func (runner *Runner) Run(ctx context.Context, req *Request) (*Result, error) {
	if runner == nil || runner.client == nil {
		return nil, fmt.Errorf("capability llm client is required")
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
	plans, stage, err := Parse(raw, req)
	attempt = newAttempt(started, raw, err)
	if err != nil {
		return &Result{Stage: stage, Attempt: attempt}, err
	}
	return &Result{Plans: plans, Stage: stage, Attempt: attempt}, nil
}
