package proposalflow

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spacehz-lab/cal/internal/core"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

// Replay is proposalflow's minimal replay JSON contract.
type Replay struct {
	Metadata     Metadata             `json:"metadata,omitempty"`
	Candidates   []caltrace.Candidate `json:"candidates"`
	ProbePlans   []ProbePlan          `json:"probe_plans"`
	proposalHash string
}

// Metadata records replay provenance without making it trusted evidence.
type Metadata struct {
	Source        string `json:"source,omitempty"`
	PromptVersion string `json:"prompt_version,omitempty"`
	Model         string `json:"model,omitempty"`
	SchemaVersion string `json:"schema_version,omitempty"`
}

// LoadReplayFile reads a proposalflow replay JSON file.
func LoadReplayFile(path string) (Replay, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Replay{}, fmt.Errorf("read proposalflow replay: %w", err)
	}
	return ParseReplay(content)
}

// ParseReplay decodes proposalflow replay JSON.
func ParseReplay(content []byte) (Replay, error) {
	var replay Replay
	if err := json.Unmarshal(content, &replay); err != nil {
		return Replay{}, fmt.Errorf("decode proposalflow replay: %w", err)
	}
	replay.proposalHash = proposalHash(content)
	if err := replay.validate(); err != nil {
		return Replay{}, err
	}
	return replay, nil
}

// Propose turns replay JSON into one complete Proposal result.
func (replay Replay) Propose(_ context.Context, request Request) (Result, error) {
	if err := replay.validate(); err != nil {
		return Result{}, err
	}
	result := Result{}
	for _, candidate := range replay.Candidates {
		candidate = normalizeCandidate(request.Provider, candidate)
		candidate = replay.withProvenance(candidate)
		result.Candidates = append(result.Candidates, candidate)
	}

	for _, replayPlan := range replay.ProbePlans {
		result.ProbePlans = append(result.ProbePlans, replayPlan)
	}
	return Select(result, SelectOptions{
		ProviderID:  request.Provider.ID,
		DebugFilter: request.DebugFilter,
	})
}

func (replay Replay) withProvenance(candidate caltrace.Candidate) caltrace.Candidate {
	source := candidate.Source
	if source == "" {
		source = "proposal:replay"
	}
	candidate.Source = source
	candidate.Provenance = &caltrace.CandidateProvenance{
		Source:        source,
		PromptVersion: replay.Metadata.PromptVersion,
		Model:         replay.Metadata.Model,
		SchemaVersion: replay.Metadata.SchemaVersion,
		ProposalHash:  replay.proposalHash,
	}
	if replay.Metadata.Source != "" && candidate.Provenance.Source == "proposal:replay" {
		candidate.Provenance.Source = "proposal:" + replay.Metadata.Source
		candidate.Source = candidate.Provenance.Source
	}
	return candidate
}

func normalizeCandidate(provider core.Provider, candidate caltrace.Candidate) caltrace.Candidate {
	if candidate.ProviderID == "" {
		candidate.ProviderID = provider.ID
	}
	return candidate
}

func (replay Replay) validate() error {
	if len(replay.Candidates) == 0 {
		return fmt.Errorf("proposalflow replay must include at least one candidate")
	}
	if len(replay.ProbePlans) == 0 {
		return fmt.Errorf("proposalflow replay must include at least one probe plan")
	}
	for index, candidate := range replay.Candidates {
		if candidate.CapabilityID == "" {
			return fmt.Errorf("proposalflow replay candidate %d capability_id is required", index)
		}
		if candidate.Description == "" {
			return fmt.Errorf("proposalflow replay candidate %d description is required", index)
		}
		if candidate.Execution.Kind == "" {
			return fmt.Errorf("proposalflow replay candidate %d execution is required", index)
		}
	}
	for index, plan := range replay.ProbePlans {
		if plan.CandidateIndex < 0 || plan.CandidateIndex >= len(replay.Candidates) {
			return fmt.Errorf("proposalflow replay probe_plan %d candidate_index %d is out of range", index, plan.CandidateIndex)
		}
		if err := core.ValidateVerifySpec(plan.Verify); err != nil {
			return fmt.Errorf("proposalflow replay probe_plan %d verify: %w", index, err)
		}
	}
	return nil
}

func proposalHash(content []byte) string {
	sum := sha256.Sum256(content)
	return fmt.Sprintf("%x", sum)
}
