package trace

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/spacehz-lab/cal/internal/core"
)

func TestNewEntryRecordsProviders(t *testing.T) {
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	record := NewEntry("trace_abc", now, []core.Provider{
		{ID: "provider_a"},
		{ID: "provider_b"},
	})
	if record.ID != "trace_abc" || record.Status != StatusCompleted {
		t.Fatalf("NewEntry() = %#v, want completed trace", record)
	}
	if len(record.ProviderIDs) != 2 || record.ProviderIDs[0] != "provider_a" || record.ProviderIDs[1] != "provider_b" {
		t.Fatalf("provider ids = %#v, want provider_a/provider_b", record.ProviderIDs)
	}
	if record.StartedAt == "" || record.EndedAt == "" {
		t.Fatalf("trace timestamps missing: %#v", record)
	}
}

func TestNewIDUsesPrefix(t *testing.T) {
	id := NewID(time.Unix(123, 0))
	if !strings.HasPrefix(id, "trace_") {
		t.Fatalf("NewID() = %q, want trace_ prefix", id)
	}
}

func TestPromotionCandidateIndexIsSerializedWhenZero(t *testing.T) {
	record := Trace{
		ID:     "trace_abc",
		Status: StatusCompleted,
		Promotions: []Promotion{{
			CandidateIndex: 0,
			CapabilityID:   "document.export_pdf",
			ProviderID:     "provider_cli",
		}},
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if !strings.Contains(string(data), `"candidate_index":0`) {
		t.Fatalf("trace JSON = %s, want explicit candidate_index=0", data)
	}
}
