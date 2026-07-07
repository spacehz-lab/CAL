package proposal

import (
	"time"

	"github.com/spacehz-lab/cal/internal/model"
)

const (
	ProposalSchemaVersion = "proposal.v1"
	ProposalPromptVersion = "proposal-cli-v1"
)

func diagnostics(modelName string, stages []model.ProposalStage, attempts []model.ProposalAttempt) *model.ProposalTrace {
	filteredStages := make([]model.ProposalStage, 0, len(stages))
	for _, stage := range stages {
		if stage.Name != "" {
			filteredStages = append(filteredStages, stage)
		}
	}
	if len(filteredStages) == 0 && len(attempts) == 0 {
		return nil
	}
	return &model.ProposalTrace{
		SchemaVersion: ProposalSchemaVersion,
		PromptVersion: ProposalPromptVersion,
		Model:         modelName,
		Stages:        filteredStages,
		Attempts:      attempts,
	}
}

func newAttempt(stage model.ProposalStageName, started time.Time, raw string, err error) model.ProposalAttempt {
	attempt := model.ProposalAttempt{
		Stage:       stage,
		Status:      model.ProposalAttemptSucceeded,
		DurationMS:  time.Since(started).Milliseconds(),
		RawResponse: raw,
	}
	if err != nil {
		attempt.Status = model.ProposalAttemptFailed
		attempt.Error = &model.RecordError{Code: CodeProposalStageFailed, Message: err.Error()}
	}
	return attempt
}

func withCapability(attempt model.ProposalAttempt, capabilityID string) model.ProposalAttempt {
	attempt.CapabilityID = capabilityID
	return attempt
}

func withCandidate(attempt model.ProposalAttempt, index int) model.ProposalAttempt {
	attempt.CandidateIndex = &index
	return attempt
}
