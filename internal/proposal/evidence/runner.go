package evidence

import (
	"context"
	"fmt"
	"time"

	"github.com/spacehz-lab/cal/internal/llm"
	"github.com/spacehz-lab/cal/internal/model"
)

// Request asks the evidence stage to build a deterministic verify spec.
type Request struct {
	Provider       *model.Provider
	Observations   []model.Observation
	Candidate      *model.Candidate
	CandidateIndex int
	Material       Material
}

// Material carries probe inputs and fixtures from the binding stage.
type Material struct {
	Inputs   map[string]any `json:"inputs,omitempty"`
	Fixtures []Fixture      `json:"fixtures,omitempty"`
}

// Fixture is one file material needed for probing.
type Fixture struct {
	Input    string `json:"input,omitempty"`
	Filename string `json:"filename,omitempty"`
	Content  string `json:"content,omitempty"`
}

// Result is the evidence stage output.
type Result struct {
	Verify  model.VerifySpec
	Stage   model.ProposalStage
	Attempt model.ProposalAttempt
}

// Runner owns the LLM-backed evidence stage.
type Runner struct {
	client llm.Client
}

// NewRunner creates an evidence stage runner.
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

// Run plans deterministic verification for one candidate.
func (runner *Runner) Run(ctx context.Context, req *Request) (*Result, error) {
	if runner == nil || runner.client == nil {
		return nil, fmt.Errorf("evidence llm client is required")
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
	verify, stage, err := Parse(raw, req)
	attempt = newAttempt(started, raw, err, req)
	if err != nil {
		return &Result{Stage: stage, Attempt: attempt}, err
	}
	return &Result{Verify: verify, Stage: stage, Attempt: attempt}, nil
}
