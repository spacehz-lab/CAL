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
			CapabilityID:   "document.convert",
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

func TestProposalDiagnosticsUseStableJSONStrings(t *testing.T) {
	candidateIndex := 0
	record := Trace{
		ID:     "trace_abc",
		Status: StatusCompleted,
		Proposal: &ProposalTrace{
			Stages: []ProposalStage{{
				Name: ProposalStageSurface,
				Summary: map[ProposalSummaryKey]int{
					ProposalSummaryRaw:      2,
					ProposalSummarySelected: 1,
				},
				Items: []ProposalItem{{
					ID:       "s1",
					Name:     "-lint",
					Decision: ProposalDecisionKeep,
					Reason:   "local_policy",
				}},
			}, {
				Name: ProposalStageBinding,
				Summary: map[ProposalSummaryKey]int{
					ProposalSummaryRaw:      1,
					ProposalSummarySelected: 1,
				},
			}},
			Attempts: []ProposalAttempt{{
				Stage:          ProposalStageEvidence,
				CapabilityID:   "document.convert",
				CandidateIndex: &candidateIndex,
				Status:         ProposalAttemptFailed,
				Error:          &core.RecordError{Code: "proposal_stage_failed", Message: "bad verify"},
				RawResponse:    `{"verify":{}}`,
			}},
		},
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	text := string(data)
	for _, want := range []string{`"name":"surface"`, `"name":"binding"`, `"raw":2`, `"selected":1`, `"decision":"keep"`, `"reason":"local_policy"`, `"attempts"`, `"stage":"evidence"`, `"candidate_index":0`, `"status":"failed"`, `"raw_response":"{\"verify\":{}}"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("trace JSON = %s, want %s", data, want)
		}
	}
}
