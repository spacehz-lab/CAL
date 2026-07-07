package binding

import (
	"context"
	"fmt"
	"time"

	"github.com/spacehz-lab/cal/internal/llm"
	"github.com/spacehz-lab/cal/internal/model"
)

// Request asks the binding stage to map one capability to provider execution candidates.
type Request struct {
	Provider      *model.Provider
	Observations  []model.Observation
	Surfaces      []SurfaceItem
	Capability    Plan
	MaxCandidates int
}

// SurfaceItem is the binding stage's view of an observed surface.
type SurfaceItem struct {
	ID          string `json:"id,omitempty"`
	Kind        string `json:"kind,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Usage       string `json:"usage,omitempty"`
}

// Plan is the provider-independent capability selected for binding.
type Plan struct {
	CapabilityID     string   `json:"capability_id"`
	Description      string   `json:"description,omitempty"`
	SourceSurfaceIDs []string `json:"source_surface_ids,omitempty"`
	Confidence       string   `json:"confidence,omitempty"`
}

// ProbeMaterial carries example inputs and fixtures for later evidence planning.
type ProbeMaterial struct {
	CandidateIndex int            `json:"candidate_index"`
	Inputs         map[string]any `json:"inputs,omitempty"`
	Fixtures       []Fixture      `json:"fixtures,omitempty"`
}

// Fixture is one file material needed for probing.
type Fixture struct {
	Input    string `json:"input,omitempty"`
	Filename string `json:"filename,omitempty"`
	Content  string `json:"content,omitempty"`
}

// Result is the binding stage output.
type Result struct {
	Candidates []model.Candidate
	Materials  []ProbeMaterial
	Stage      model.ProposalStage
	Attempt    model.ProposalAttempt
}

// Runner owns the LLM-backed binding stage.
type Runner struct {
	client llm.Client
}

// NewRunner creates a binding stage runner.
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

// Run plans concrete execution candidates for one capability.
func (runner *Runner) Run(ctx context.Context, req *Request) (*Result, error) {
	if runner == nil || runner.client == nil {
		return nil, fmt.Errorf("binding llm client is required")
	}
	started := time.Now()
	response, err := runner.client.Complete(ctx, prompt(req))
	raw := ""
	if response != nil {
		raw = response.Text
	}
	attempt := newAttempt(started, raw, err, req)
	if err != nil {
		return &Result{Attempt: attempt}, err
	}
	candidates, materials, stage, err := Parse(raw, req)
	attempt = newAttempt(started, raw, err, req)
	if err != nil {
		return &Result{Stage: stage, Attempt: attempt}, err
	}
	return &Result{Candidates: candidates, Materials: materials, Stage: stage, Attempt: attempt}, nil
}
