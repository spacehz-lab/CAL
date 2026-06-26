package llm

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	sharedllm "github.com/spacehz-lab/cal/internal/llm"
	"github.com/spacehz-lab/cal/internal/proposal"
)

const (
	proposalPromptVersion = "prompt-v1"
	proposalSchemaVersion = "proposal.v1"
	proposalSource        = "llm"

	logKeyProviderID             = "provider_id"
	logKeyProviderKind           = "provider_kind"
	logKeyModel                  = "model"
	logKeyPromptVersion          = "prompt_version"
	logKeySchemaVersion          = "schema_version"
	logKeyCandidateCount         = "candidate_count"
	logKeyProposalCandidateCount = "proposal_candidate_count"
	logKeyProbePlanCount         = "probe_plan_count"
	logKeyVerifierPackageCount   = "verifier_package_count"
	logKeyDurationMS             = "duration_ms"
)

// Proposer adapts an llm client to candidate and probe-plan interfaces.
type Proposer struct {
	client       sharedllm.Client
	materializer proposal.Materializer
	loaded       bool
}

type modelReporter interface {
	Model() string
}

// NewProposer builds an llm-backed candidate and probe-plan proposer.
func NewProposer(client sharedllm.Client) *Proposer {
	return &Proposer{client: client}
}

// Propose sends a bounded prompt and parses the returned proposal.
func (proposer *Proposer) Propose(ctx context.Context, request proposal.Request) (proposal.Response, error) {
	started := time.Now()
	proposer.logStarted(request)
	if err := proposer.loadProposal(ctx, request, started); err != nil {
		return proposal.Response{}, err
	}
	response, err := proposer.materializer.Propose(ctx, request)
	if err != nil {
		proposer.logFailed(request, "select", err, 0, started)
		return proposal.Response{}, err
	}
	proposer.logCompleted(request, response, started)
	return response, nil
}

// Plan returns the probe plan from the loaded proposal.
func (proposer *Proposer) Plan(ctx context.Context, request proposal.ProbePlanRequest) (proposal.ProbePlan, error) {
	if !proposer.loaded {
		return proposal.ProbePlan{}, ErrNoProposal
	}
	return proposer.materializer.Plan(ctx, request)
}

func (proposer *Proposer) loadProposal(ctx context.Context, request proposal.Request, started time.Time) error {
	if proposer.client == nil {
		proposer.logFailed(request, "client", sharedllm.ErrNoClient, 0, started)
		return sharedllm.ErrNoClient
	}
	content, err := proposer.client.Complete(ctx, newPromptBuilder().Build(request))
	if err != nil {
		proposer.logFailed(request, "complete", err, 0, started)
		return err
	}
	materializer, err := proposal.ParseWithMetadata(content, proposer.metadata())
	if err != nil {
		proposer.logFailed(request, "parse", err, len(content), started)
		return err
	}
	proposer.materializer = materializer
	proposer.loaded = true
	return nil
}

func (proposer *Proposer) metadata() proposal.Metadata {
	metadata := proposal.Metadata{
		Source:        proposalSource,
		PromptVersion: proposalPromptVersion,
		SchemaVersion: proposalSchemaVersion,
	}
	if client, ok := proposer.client.(modelReporter); ok {
		metadata.Model = strings.TrimSpace(client.Model())
	}
	return metadata
}

func (proposer *Proposer) logStarted(request proposal.Request) {
	metadata := proposer.metadata()
	attrs := proposer.logAttrs(request, metadata,
		"observation_count", len(request.Observations),
		"existing_capability_count", len(request.ExistingCapabilityIDs),
	)
	slog.Info("proposal llm started", attrs...)
}

func (proposer *Proposer) logCompleted(request proposal.Request, response proposal.Response, started time.Time) {
	metadata := proposer.metadata()
	stats := proposer.materializer.Stats()
	attrs := proposer.logAttrs(request, metadata,
		logKeyCandidateCount, len(response.Candidates),
		logKeyProposalCandidateCount, stats.CandidateCount,
		logKeyProbePlanCount, stats.ProbePlanCount,
		logKeyVerifierPackageCount, stats.VerifierPackageCount,
		logKeyDurationMS, time.Since(started).Milliseconds(),
	)
	slog.Info("proposal llm completed", attrs...)
}

func (proposer *Proposer) logFailed(request proposal.Request, stage string, err error, responseBytes int, started time.Time) {
	metadata := proposer.metadata()
	attrs := proposer.logAttrs(request, metadata,
		"stage", stage,
		logKeyDurationMS, time.Since(started).Milliseconds(),
	)
	if err != nil {
		attrs = append(attrs, "error", err.Error())
	}
	if responseBytes > 0 {
		attrs = append(attrs, "response_bytes", responseBytes)
	}
	var invalid proposal.InvalidProposalError
	if errors.As(err, &invalid) {
		attrs = append(attrs,
			"proposal_hash", invalid.ProposalHash,
			logKeyProposalCandidateCount, invalid.Stats.CandidateCount,
			logKeyProbePlanCount, invalid.Stats.ProbePlanCount,
			logKeyVerifierPackageCount, invalid.Stats.VerifierPackageCount,
		)
	}
	slog.Warn("proposal llm failed", attrs...)
}

func (proposer *Proposer) logAttrs(request proposal.Request, metadata proposal.Metadata, attrs ...any) []any {
	base := []any{
		logKeyProviderID, request.Provider.ID,
		logKeyProviderKind, request.Provider.Kind,
		logKeyModel, metadata.Model,
		logKeyPromptVersion, metadata.PromptVersion,
		logKeySchemaVersion, metadata.SchemaVersion,
	}
	return append(base, attrs...)
}
