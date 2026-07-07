package proposal

import "time"

const (
	DefaultTimeout              = 10 * time.Minute
	DefaultPerCapabilityTimeout = 5 * time.Minute
	DefaultMaxSurfaceItems      = 40
	DefaultConcurrency          = 20
	MaxConcurrency              = 50
)

// Options controls proposal runtime bounds.
type Options struct {
	Timeout              time.Duration
	PerCapabilityTimeout time.Duration
	SurfaceLimit         int
	Concurrency          int
	CandidateLimit       int
}

func normalizeOptions(options Options) Options {
	if options.Timeout <= 0 {
		options.Timeout = DefaultTimeout
	}
	if options.PerCapabilityTimeout <= 0 {
		options.PerCapabilityTimeout = DefaultPerCapabilityTimeout
	}
	if options.SurfaceLimit <= 0 {
		options.SurfaceLimit = DefaultMaxSurfaceItems
	}
	if options.Concurrency <= 0 {
		options.Concurrency = DefaultConcurrency
	}
	if options.Concurrency > MaxConcurrency {
		options.Concurrency = MaxConcurrency
	}
	if options.CandidateLimit < 0 {
		options.CandidateLimit = 0
	}
	return options
}
