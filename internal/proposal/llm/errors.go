package llm

import "errors"

// ErrNoProposal reports probe planning before proposal generation loaded a proposal.
var ErrNoProposal = errors.New("llm proposal is not loaded")
