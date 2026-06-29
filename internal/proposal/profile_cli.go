package proposal

import (
	"time"
)

const (
	cliPromptVersion      = "proposal-cli-v1"
	cliProposalSource     = "llm"
	cliProposalSchema     = "proposal.v1"
	defaultMaxSurface     = 40
	defaultMaxCapability  = 12
	defaultMaxCandidate   = 2
	defaultConcurrency    = 50
	defaultBindingTimeout = 4 * time.Minute
)

func cliProfile() profile {
	return profile{
		id:                         "cli",
		maxSurfaceItems:            defaultMaxSurface,
		maxCapabilities:            defaultMaxCapability,
		maxCandidatesPerCapability: defaultMaxCandidate,
		bindingTimeout:             defaultBindingTimeout,
		concurrency:                defaultConcurrency,
	}
}
