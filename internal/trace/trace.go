package trace

import (
	"strconv"
	"time"

	"github.com/spacehz-lab/cal/internal/core"
)

// NewID derives a local trace id from a timestamp.
func NewID(now time.Time) string {
	return "trace_" + strconv.FormatInt(now.UTC().UnixNano(), 10)
}

// NewEntry builds a completed Trace for an Entry-only discovery job.
func NewEntry(id string, now time.Time, providers []core.Provider) Trace {
	providerIDs := make([]string, 0, len(providers))
	for _, provider := range providers {
		providerIDs = append(providerIDs, provider.ID)
	}
	return Trace{
		ID:          id,
		StartedAt:   now.UTC().Format(time.RFC3339Nano),
		EndedAt:     now.UTC().Format(time.RFC3339Nano),
		Status:      StatusCompleted,
		ProviderIDs: providerIDs,
	}
}
