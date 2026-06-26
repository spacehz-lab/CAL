package discovery

import "fmt"

// CodedError is a stable domain error that control surfaces can render.
type CodedError struct {
	Code    string
	Message string
}

const (
	CodeObserverUnavailable     = "observer_unavailable"
	CodeProposerUnavailable     = "proposer_unavailable"
	CodeProbePlannerUnavailable = "probe_planner_unavailable"
	CodeProviderNotFound        = "provider_not_found"
	CodeUnsupportedProviderKind = "unsupported_provider_kind"
	CodeObservationFailed       = "observation_failed"
	CodeCandidateProposalFailed = "candidate_proposal_failed"
	CodeCandidateNotFound       = "candidate_not_found"
	CodeVerificationFailed      = "verification_failed"
	CodePromotionFailed         = "promotion_failed"
)

func (err CodedError) Error() string {
	return err.Message
}

func newCodedError(code, message string) CodedError {
	return CodedError{Code: code, Message: message}
}

func newCodedErrorf(code, format string, args ...any) CodedError {
	return newCodedError(code, fmt.Sprintf(format, args...))
}
