package proposal

import "time"

type profile struct {
	id                         string
	maxSurfaceItems            int
	maxCapabilities            int
	maxCandidatesPerCapability int
	bindingTimeout             time.Duration
	concurrency                int
}

func selectProfile(Request) profile {
	return cliProfile()
}
