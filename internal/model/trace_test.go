package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPromotionCandidateIndexIsSerializedWhenZero(t *testing.T) {
	record := Trace{
		ID:     "trace_abc",
		Status: TraceStatusCompleted,
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
		Status: TraceStatusCompleted,
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
				Error:          &RecordError{Code: "proposal_stage_failed", Message: "bad verify"},
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
