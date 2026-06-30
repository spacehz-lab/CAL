package proposal

import (
	"time"

	"github.com/spacehz-lab/cal/internal/core"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

const proposalAttemptErrorCode = "proposal_stage_failed"

func newAttempt(stage caltrace.ProposalStageName, started time.Time, raw []byte, err error) caltrace.ProposalAttempt {
	attempt := caltrace.ProposalAttempt{
		Stage:      stage,
		Status:     caltrace.ProposalAttemptSucceeded,
		DurationMS: time.Since(started).Milliseconds(),
	}
	if len(raw) > 0 {
		attempt.RawResponse = string(raw)
	}
	if err != nil {
		attempt.Status = caltrace.ProposalAttemptFailed
		attempt.Error = &core.RecordError{
			Code:    proposalAttemptErrorCode,
			Message: err.Error(),
		}
	}
	return attempt
}

func withAttemptCapability(attempt caltrace.ProposalAttempt, capabilityID string) caltrace.ProposalAttempt {
	attempt.CapabilityID = capabilityID
	return attempt
}

func withAttemptCandidate(attempt caltrace.ProposalAttempt, index int) caltrace.ProposalAttempt {
	attempt.CandidateIndex = &index
	return attempt
}
