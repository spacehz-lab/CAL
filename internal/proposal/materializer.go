package proposal

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/runtime"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

const defaultSource = "proposal:replay"

// Materializer adapts one proposal into candidate and probe-plan providers.
type Materializer struct {
	proposal     Proposal
	proposalHash string
}

// Stats contains safe proposal counts for diagnostics.
type Stats struct {
	CandidateCount       int
	ProbePlanCount       int
	VerifierPackageCount int
}

// InvalidProposalError reports a proposal contract failure with safe diagnostics.
type InvalidProposalError struct {
	Message      string
	Stats        Stats
	ProposalHash string
}

func (err InvalidProposalError) Error() string {
	return err.Message
}

// LoadFile reads a replayed proposal JSON file.
func LoadFile(path string) (Materializer, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Materializer{}, fmt.Errorf("read proposal: %w", err)
	}
	return Parse(content)
}

// Parse decodes a replayed proposal JSON document.
func Parse(content []byte) (Materializer, error) {
	var proposal Proposal
	if err := json.Unmarshal(content, &proposal); err != nil {
		return Materializer{}, fmt.Errorf("decode proposal: %w", err)
	}
	return newMaterializer(content, proposal)
}

// ParseWithMetadata decodes proposal JSON using CAL-owned metadata.
//
// This is for live adapters where provenance is an adapter fact, not model output.
func ParseWithMetadata(content []byte, metadata Metadata) (Materializer, error) {
	var proposal Proposal
	if err := json.Unmarshal(content, &proposal); err != nil {
		return Materializer{}, fmt.Errorf("decode proposal: %w", err)
	}
	proposal.Metadata = metadata
	for index := range proposal.Candidates {
		proposal.Candidates[index].Source = ""
	}
	return newMaterializer(content, proposal)
}

func newMaterializer(content []byte, proposal Proposal) (Materializer, error) {
	proposalHash := proposalContentHash(content)
	stats := proposalStats(proposal)
	if len(proposal.Candidates) == 0 {
		return Materializer{}, newInvalidProposalError("proposal must include at least one candidate", stats, proposalHash)
	}
	if len(proposal.ProbePlans) == 0 {
		return Materializer{}, newInvalidProposalError("proposal must include at least one probe plan", stats, proposalHash)
	}
	if err := validateVerifierPackageIDs(proposal.VerifierPackages); err != nil {
		return Materializer{}, newInvalidProposalError(err.Error(), stats, proposalHash)
	}
	for index, candidate := range proposal.Candidates {
		if strings.TrimSpace(candidate.Description) == "" {
			return Materializer{}, newInvalidProposalError(fmt.Sprintf("proposal candidate %d description is required", index), stats, proposalHash)
		}
	}
	materializer := Materializer{proposal: proposal, proposalHash: proposalHash}
	slog.Info("proposal materializer loaded",
		"source", materializer.proposal.Metadata.Source,
		"model", materializer.proposal.Metadata.Model,
		"prompt_version", materializer.proposal.Metadata.PromptVersion,
		"schema_version", materializer.proposal.Metadata.SchemaVersion,
		"proposal_hash", materializer.proposalHash,
		"candidate_count", stats.CandidateCount,
		"probe_plan_count", stats.ProbePlanCount,
		"verifier_package_count", stats.VerifierPackageCount,
	)
	return materializer, nil
}

// Stats returns safe proposal counts for logging and diagnostics.
func (materializer Materializer) Stats() Stats {
	return proposalStats(materializer.proposal)
}

func proposalStats(proposal Proposal) Stats {
	return Stats{
		CandidateCount:       len(proposal.Candidates),
		ProbePlanCount:       len(proposal.ProbePlans),
		VerifierPackageCount: len(proposal.VerifierPackages),
	}
}

func validateVerifierPackageIDs(packages []runtime.GeneratedVerifierPackage) error {
	seen := map[string]struct{}{}
	for index, pkg := range packages {
		if !core.ValidVerifierID(pkg.ID) {
			return fmt.Errorf("proposal verifier package %d id %q is invalid", index, pkg.ID)
		}
		if strings.HasPrefix(pkg.ID, "verifier_") {
			return fmt.Errorf("proposal verifier package id %q must not start with verifier_", pkg.ID)
		}
		if _, ok := seen[pkg.ID]; ok {
			return fmt.Errorf("proposal verifier package id %q is duplicated", pkg.ID)
		}
		seen[pkg.ID] = struct{}{}
	}
	return nil
}

func proposalContentHash(content []byte) string {
	sum := sha256.Sum256(content)
	return fmt.Sprintf("%x", sum)
}

func newInvalidProposalError(message string, stats Stats, proposalHash string) InvalidProposalError {
	return InvalidProposalError{
		Message:      message,
		Stats:        stats,
		ProposalHash: proposalHash,
	}
}

// Propose returns proposal candidates for the requested provider and optional hint.
func (materializer Materializer) Propose(_ context.Context, request Request) (Response, error) {
	candidates := make([]caltrace.Candidate, 0, len(materializer.proposal.Candidates))
	for _, candidate := range materializer.proposal.Candidates {
		providerID := candidate.ProviderID
		if providerID == "" {
			providerID = request.Provider.ID
		}
		if providerID != request.Provider.ID {
			continue
		}
		if request.Hint != "" && candidate.CapabilityID != request.Hint {
			continue
		}
		source := materializer.source(candidate.Source)
		candidates = append(candidates, caltrace.Candidate{
			ProviderID:       providerID,
			CapabilityID:     candidate.CapabilityID,
			Description:      candidate.Description,
			Source:           source,
			Provenance:       materializer.provenance(source),
			InputConstraints: candidate.InputConstraints,
			Execution:        candidate.Execution,
		})
	}
	slog.Info("proposal candidates selected",
		"provider_id", request.Provider.ID,
		"capability_hint", request.Hint,
		"candidate_count", len(candidates),
		"proposal_hash", materializer.proposalHash,
	)
	return Response{Candidates: candidates}, nil
}

// Plan materializes the proposal probe plan for one candidate.
func (materializer Materializer) Plan(_ context.Context, request ProbePlanRequest) (ProbePlan, error) {
	started := time.Now()
	if request.WorkDir == "" {
		materializer.logPlanFailed(request, "workdir", started)
		return ProbePlan{}, fmt.Errorf("probe work directory is required")
	}
	index, err := materializer.candidateIndex(request.Candidate)
	if err != nil {
		materializer.logPlanFailed(request, "candidate_match", started)
		return ProbePlan{}, err
	}
	for planIndex, plan := range materializer.proposal.ProbePlans {
		if plan.CandidateIndex != index {
			continue
		}
		probePlan, err := newProbeMaterializer(
			request.WorkDir,
			materializer.proposalHash,
			index,
			planIndex,
			materializer.proposal.VerifierPackages,
		).materialize(plan)
		if err != nil {
			materializer.logPlanFailed(request, "materialize", started)
			return ProbePlan{}, err
		}
		if err := validateProbeExecutionInputs(request.Candidate.Execution, probePlan.Inputs); err != nil {
			materializer.logPlanFailed(request, "contract", started)
			return ProbePlan{}, err
		}
		materializer.logPlanCompleted(request, probePlan, len(plan.Fixtures), started)
		return probePlan, nil
	}
	materializer.logPlanFailed(request, "probe_plan", started)
	return ProbePlan{}, fmt.Errorf("proposal has no probe plan for candidate %d", index)
}

func validateProbeExecutionInputs(execution core.Execution, inputs map[string]any) error {
	if execution.Kind != core.ExecutionKindCLI {
		return nil
	}
	args, err := proposalExecutionArgs(execution)
	if err != nil {
		return err
	}
	for _, name := range missingExecutionTemplateInputs(args, inputs) {
		return fmt.Errorf("proposal probe inputs missing execution template input %q", name)
	}
	stdoutPathInput, hasStdoutPathInput, err := proposalStdoutPathInput(execution)
	if err != nil {
		return err
	}
	if hasStdoutPathInput {
		if _, ok := inputs[stdoutPathInput]; !ok {
			return fmt.Errorf("proposal probe inputs missing stdout path input %q", stdoutPathInput)
		}
	}
	if _, ok := inputs["target"]; ok && !argsUseInput(args, "target") && (!hasStdoutPathInput || stdoutPathInput != "target") {
		return fmt.Errorf("proposal probe target input is not produced by execution args or stdout_path_input")
	}
	return nil
}

func proposalExecutionArgs(execution core.Execution) ([]string, error) {
	value, ok := execution.Spec[core.ExecutionSpecArgs]
	if !ok {
		return nil, fmt.Errorf("proposal cli execution args are required")
	}
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...), nil
	case []any:
		args := make([]string, len(typed))
		for index, item := range typed {
			arg, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("proposal cli execution args must be strings")
			}
			args[index] = arg
		}
		return args, nil
	default:
		return nil, fmt.Errorf("proposal cli execution args must be an array")
	}
}

func proposalStdoutPathInput(execution core.Execution) (string, bool, error) {
	value, ok := execution.Spec[core.ExecutionSpecStdoutPathInput]
	if !ok {
		return "", false, nil
	}
	input, ok := value.(string)
	if !ok || strings.TrimSpace(input) == "" {
		return "", false, fmt.Errorf("proposal cli execution stdout_path_input must be a string")
	}
	return input, true, nil
}

func missingExecutionTemplateInputs(args []string, inputs map[string]any) []string {
	missing := []string{}
	seen := map[string]struct{}{}
	for _, arg := range args {
		for _, name := range executionTemplateInputs(arg) {
			if _, ok := inputs[name]; ok {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			missing = append(missing, name)
		}
	}
	return missing
}

func argsUseInput(args []string, target string) bool {
	for _, arg := range args {
		for _, name := range executionTemplateInputs(arg) {
			if name == target {
				return true
			}
		}
	}
	return false
}

func executionTemplateInputs(arg string) []string {
	names := []string{}
	for {
		start := strings.Index(arg, "{{")
		if start < 0 {
			return names
		}
		rest := arg[start+2:]
		end := strings.Index(rest, "}}")
		if end < 0 {
			return names
		}
		name := strings.TrimSpace(rest[:end])
		if name != "" {
			names = append(names, name)
		}
		arg = rest[end+2:]
	}
}

func (materializer Materializer) candidateIndex(candidate caltrace.Candidate) (int, error) {
	for index, proposed := range materializer.proposal.Candidates {
		providerID := proposed.ProviderID
		if providerID == "" {
			providerID = candidate.ProviderID
		}
		if providerID == candidate.ProviderID &&
			proposed.CapabilityID == candidate.CapabilityID &&
			reflect.DeepEqual(proposed.Execution, candidate.Execution) {
			return index, nil
		}
	}
	return -1, fmt.Errorf("proposal has no probe plan candidate match for %q", candidate.CapabilityID)
}

func (materializer Materializer) source(candidateSource string) string {
	if candidateSource != "" {
		return candidateSource
	}
	if materializer.proposal.Metadata.Source != "" {
		return "proposal:" + materializer.proposal.Metadata.Source
	}
	return defaultSource
}

func (materializer Materializer) provenance(source string) *caltrace.CandidateProvenance {
	return &caltrace.CandidateProvenance{
		Source:        source,
		PromptVersion: materializer.proposal.Metadata.PromptVersion,
		Model:         materializer.proposal.Metadata.Model,
		SchemaVersion: materializer.proposal.Metadata.SchemaVersion,
		ProposalHash:  materializer.proposalHash,
	}
}

func (materializer Materializer) logPlanCompleted(request ProbePlanRequest, plan ProbePlan, fixtureCount int, started time.Time) {
	slog.Info("proposal probe plan materialized",
		"provider_id", request.Candidate.ProviderID,
		"capability_id", request.Candidate.CapabilityID,
		"verifier_id", plan.Verifier.ID,
		"input_count", len(plan.Inputs),
		"fixture_count", fixtureCount,
		"proposal_hash", materializer.proposalHash,
		"duration_ms", time.Since(started).Milliseconds(),
	)
}

func (materializer Materializer) logPlanFailed(request ProbePlanRequest, stage string, started time.Time) {
	slog.Warn("proposal probe plan failed",
		"provider_id", request.Candidate.ProviderID,
		"capability_id", request.Candidate.CapabilityID,
		"stage", stage,
		"proposal_hash", materializer.proposalHash,
		"duration_ms", time.Since(started).Milliseconds(),
	)
}
