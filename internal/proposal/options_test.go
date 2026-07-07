package proposal

import "testing"

func TestNormalizeOptionsAppliesDefaultsAndCapsConcurrency(t *testing.T) {
	options := normalizeOptions(Options{Concurrency: MaxConcurrency + 1, CandidateLimit: -1})

	if options.Timeout != DefaultTimeout {
		t.Fatalf("Timeout = %v, want %v", options.Timeout, DefaultTimeout)
	}
	if options.PerCapabilityTimeout != DefaultPerCapabilityTimeout {
		t.Fatalf("PerCapabilityTimeout = %v, want %v", options.PerCapabilityTimeout, DefaultPerCapabilityTimeout)
	}
	if options.SurfaceLimit != DefaultMaxSurfaceItems {
		t.Fatalf("SurfaceLimit = %d, want %d", options.SurfaceLimit, DefaultMaxSurfaceItems)
	}
	if options.Concurrency != MaxConcurrency {
		t.Fatalf("Concurrency = %d, want %d", options.Concurrency, MaxConcurrency)
	}
	if options.CandidateLimit != 0 {
		t.Fatalf("CandidateLimit = %d, want 0", options.CandidateLimit)
	}
}
