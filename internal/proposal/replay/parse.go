package replay

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/proposal"
)

const (
	sourceReplay          = "proposal:replay"
	defaultMetadataSource = "replay"
)

type replayFile struct {
	Metadata   metadata             `json:"metadata"`
	Candidates []model.Candidate    `json:"candidates"`
	ProbePlans []proposal.ProbePlan `json:"probe_plans"`
}

type metadata struct {
	Source        string `json:"source"`
	PromptVersion string `json:"prompt_version"`
	Model         string `json:"model"`
	SchemaVersion string `json:"schema_version"`
}

func parse(content []byte, providerID string) (*proposal.Result, error) {
	var file replayFile
	if err := json.Unmarshal(content, &file); err != nil {
		return nil, fmt.Errorf("decode replay proposal: %w", err)
	}
	if err := validateFile(&file); err != nil {
		return nil, err
	}
	hash := proposalHash(content)
	candidates := make([]model.Candidate, 0, len(file.Candidates))
	for index, candidate := range file.Candidates {
		normalized, err := normalizeCandidate(candidate, providerID, file.Metadata, hash)
		if err != nil {
			return nil, fmt.Errorf("candidate %d: %w", index, err)
		}
		candidates = append(candidates, normalized)
	}
	plans := make([]proposal.ProbePlan, 0, len(file.ProbePlans))
	for index, plan := range file.ProbePlans {
		if err := validateProbePlan(plan, len(file.Candidates)); err != nil {
			return nil, fmt.Errorf("probe_plan %d: %w", index, err)
		}
		plans = append(plans, plan)
	}
	return &proposal.Result{
		Candidates:  candidates,
		ProbePlans:  plans,
		Diagnostics: diagnostics(file.Metadata, hash, candidates),
	}, nil
}

func validateFile(file *replayFile) error {
	if file == nil {
		return fmt.Errorf("replay proposal is required")
	}
	schema := strings.TrimSpace(file.Metadata.SchemaVersion)
	if schema != "" && schema != proposal.ProposalSchemaVersion {
		return fmt.Errorf("unsupported replay schema version %q", schema)
	}
	if len(file.Candidates) == 0 {
		return fmt.Errorf("replay proposal has no candidates")
	}
	if len(file.ProbePlans) == 0 {
		return fmt.Errorf("replay proposal has no probe plans")
	}
	return nil
}

func normalizeCandidate(candidate model.Candidate, providerID string, meta metadata, hash string) (model.Candidate, error) {
	candidate.ProviderID = strings.TrimSpace(providerID)
	candidate.CapabilityID = strings.TrimSpace(candidate.CapabilityID)
	candidate.Description = strings.TrimSpace(candidate.Description)
	candidate.Source = sourceReplay
	candidate.Execution.Kind = model.ExecutionKind(strings.TrimSpace(string(candidate.Execution.Kind)))
	candidate.Provenance = &model.CandidateProvenance{
		Source:        metadataSource(meta),
		PromptVersion: strings.TrimSpace(meta.PromptVersion),
		Model:         strings.TrimSpace(meta.Model),
		SchemaVersion: strings.TrimSpace(meta.SchemaVersion),
		ProposalHash:  hash,
	}
	switch {
	case candidate.ProviderID == "":
		return model.Candidate{}, fmt.Errorf("provider id is required")
	case !model.ValidCapabilityID(candidate.CapabilityID):
		return model.Candidate{}, fmt.Errorf("capability id %q is invalid", candidate.CapabilityID)
	case candidate.Description == "":
		return model.Candidate{}, fmt.Errorf("description is required")
	case !validExecutionKind(candidate.Execution.Kind):
		return model.Candidate{}, fmt.Errorf("execution kind %q is invalid", candidate.Execution.Kind)
	}
	if _, err := model.CanonicalExecution(candidate.Execution); err != nil {
		return model.Candidate{}, err
	}
	return candidate, nil
}

func validateProbePlan(plan proposal.ProbePlan, candidates int) error {
	switch {
	case plan.CandidateIndex < 0 || plan.CandidateIndex >= candidates:
		return fmt.Errorf("candidate_index %d is out of range", plan.CandidateIndex)
	case !validVerifyLevel(plan.Verify.Level):
		return fmt.Errorf("verify level %q is invalid", plan.Verify.Level)
	case !validVerifyMethod(plan.Verify.Method):
		return fmt.Errorf("verify method %q is invalid", plan.Verify.Method)
	case plan.Verify.Method == model.VerifyMethodExecute && len(plan.Verify.Checks) == 0:
		return fmt.Errorf("execute verify requires checks")
	}
	return nil
}

func diagnostics(meta metadata, hash string, candidates []model.Candidate) *model.ProposalTrace {
	stage := model.ProposalStage{
		Name: model.ProposalStageBinding,
		Summary: map[model.ProposalSummaryKey]int{
			model.ProposalSummaryRaw:      len(candidates),
			model.ProposalSummaryKeep:     len(candidates),
			model.ProposalSummarySelected: len(candidates),
		},
		Items: make([]model.ProposalItem, 0, len(candidates)),
	}
	for _, candidate := range candidates {
		stage.Items = append(stage.Items, model.ProposalItem{
			ID:       candidate.CapabilityID,
			Kind:     string(candidate.Execution.Kind),
			Name:     candidate.Description,
			Decision: model.ProposalDecisionKeep,
			Reason:   sourceReplay,
		})
	}
	return &model.ProposalTrace{
		SchemaVersion: metadataSchema(meta),
		PromptVersion: strings.TrimSpace(meta.PromptVersion),
		Model:         strings.TrimSpace(meta.Model),
		Stages:        []model.ProposalStage{stage},
		Attempts: []model.ProposalAttempt{{
			Stage:       model.ProposalStageBinding,
			Status:      model.ProposalAttemptSucceeded,
			RawResponse: hash,
		}},
	}
}

func metadataSource(meta metadata) string {
	if source := strings.TrimSpace(meta.Source); source != "" {
		return source
	}
	return defaultMetadataSource
}

func metadataSchema(meta metadata) string {
	if schema := strings.TrimSpace(meta.SchemaVersion); schema != "" {
		return schema
	}
	return proposal.ProposalSchemaVersion
}

func proposalHash(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func validExecutionKind(kind model.ExecutionKind) bool {
	switch kind {
	case model.ExecutionKindCLI, model.ExecutionKindMenu, model.ExecutionKindAXAction, model.ExecutionKindURLOpen:
		return true
	default:
		return false
	}
}

func validVerifyLevel(level model.VerifyLevel) bool {
	switch level {
	case model.VerifyLevelL1, model.VerifyLevelL2, model.VerifyLevelL3:
		return true
	default:
		return false
	}
}

func validVerifyMethod(method model.VerifyMethod) bool {
	switch method {
	case model.VerifyMethodContract, model.VerifyMethodExecute:
		return true
	default:
		return false
	}
}
