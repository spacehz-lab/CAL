package proposalflow

type profile struct {
	id              string
	maxSurfaceItems int
	maxCapabilities int
	concurrency     int
}

func selectProfile(Request) profile {
	return cliProfile()
}
