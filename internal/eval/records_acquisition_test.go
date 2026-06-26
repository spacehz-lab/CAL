package eval

import (
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestRecordsAcquisitionCountsMultiCandidateTrace(t *testing.T) {
	metrics := records{
		traces: []caltrace.Trace{{
			ID:     "trace_multi",
			Status: caltrace.StatusCompleted,
			Candidates: []caltrace.Candidate{
				{
					ProviderID:   "provider_fake",
					CapabilityID: "document.export_pdf",
					Source:       "proposal:multi",
					Execution:    core.Execution{Kind: core.ExecutionKindCLI},
				},
				{
					ProviderID:   "provider_fake",
					CapabilityID: "image.resize",
					Source:       "proposal:multi",
					Execution:    core.Execution{Kind: core.ExecutionKindCLI},
				},
			},
			Probes: []caltrace.Probe{
				{
					CandidateIndex: 0,
					Passed:         true,
					Verifier:       core.Verifier{ID: "file_parse_pdf"},
					Evidence:       []core.EvidenceRef{{ID: "evidence_pdf"}},
				},
				{
					CandidateIndex: 1,
					Passed:         false,
					Verifier:       core.Verifier{ID: "image_dimensions_match"},
					Error:          &core.RecordError{Code: "verification_failed", Message: "wrong size"},
				},
			},
			Promotions: []caltrace.Promotion{{
				CandidateIndex:   0,
				CapabilityID:     "document.export_pdf",
				BindingID:        "binding_pdf",
				ProviderID:       "provider_fake",
				CapabilityAction: "created",
				BindingAction:    "created",
			}},
		}},
	}.acquisition()

	if metrics.AttemptCount != 1 || metrics.CompletedCount != 1 || metrics.PromotionCount != 1 {
		t.Fatalf("acquisition counts = %#v, want one completed trace with one promotion", metrics)
	}
	if metrics.CandidateCount != 2 || metrics.ProbeCount != 2 || metrics.ProbePassCount != 1 || metrics.ProbeFailCount != 1 {
		t.Fatalf("candidate/probe counts = %#v, want two candidates with one passed probe", metrics)
	}
	if metrics.BindingPromotionRate != 0.5 || metrics.ProbeSuccessRate != 0.5 {
		t.Fatalf("rates = %#v, want half promotion and probe success rates", metrics)
	}
	if len(metrics.ByCapability) != 2 {
		t.Fatalf("by capability = %#v, want two capability buckets", metrics.ByCapability)
	}
	document := metrics.ByCapability[0]
	image := metrics.ByCapability[1]
	if document.CapabilityID != "document.export_pdf" || document.Attempts != 1 || document.Completed != 1 || document.Promotions != 1 || document.ProbePasses != 1 {
		t.Fatalf("document bucket = %#v, want promoted document capability", document)
	}
	if image.CapabilityID != "image.resize" || image.Attempts != 1 || image.Failed != 1 || image.Promotions != 0 || image.ProbeFailures != 1 {
		t.Fatalf("image bucket = %#v, want failed image candidate", image)
	}
	if len(metrics.BySource) != 1 || metrics.BySource[0].Attempts != 2 || metrics.BySource[0].Completed != 1 || metrics.BySource[0].Failed != 1 || metrics.BySource[0].Promotions != 1 {
		t.Fatalf("by source = %#v, want per-candidate source counts", metrics.BySource)
	}
}

func TestRecordsAcquisitionCountsTraceWithoutCandidates(t *testing.T) {
	metrics := records{
		traces: []caltrace.Trace{
			{ID: "entry_trace", Status: caltrace.StatusCompleted},
			{
				ID:     "trace_no_candidates",
				Status: caltrace.StatusFailed,
				Hint:   "document.export_pdf",
				Probes: []caltrace.Probe{
					{CandidateIndex: -1, Passed: true, Verifier: core.Verifier{ID: "file_exists"}},
					{CandidateIndex: -1, Passed: false, Verifier: core.Verifier{ID: "file_parse_pdf"}},
				},
				Promotions: []caltrace.Promotion{
					{CapabilityID: "document.export_pdf", ProviderID: "provider_fake"},
					{CapabilityID: "image.resize", ProviderID: "provider_fake"},
				},
			},
		},
	}.acquisition()

	if metrics.AttemptCount != 1 || metrics.FailedCount != 1 || metrics.PromotionCount != 2 {
		t.Fatalf("acquisition = %#v, want one failed acquisition attempt with two promotions", metrics)
	}
	if metrics.CandidateCount != 0 || metrics.ProbeCount != 2 || metrics.ProbePassCount != 1 || metrics.ProbeFailCount != 1 {
		t.Fatalf("probe counts = %#v, want no candidates with one pass and one fail", metrics)
	}
	if len(metrics.ByCapability) != 2 {
		t.Fatalf("by capability = %#v, want primary and promoted capability buckets", metrics.ByCapability)
	}
	if metrics.ByCapability[0].CapabilityID != "document.export_pdf" || metrics.ByCapability[0].Attempts != 1 || metrics.ByCapability[0].Failed != 1 || metrics.ByCapability[0].Promotions != 2 || metrics.ByCapability[0].Probes != 2 {
		t.Fatalf("primary capability bucket = %#v, want trace-level counts", metrics.ByCapability[0])
	}
	if metrics.ByCapability[1].CapabilityID != "image.resize" || metrics.ByCapability[1].Promotions != 1 {
		t.Fatalf("secondary capability bucket = %#v, want promotion count", metrics.ByCapability[1])
	}
	if len(metrics.BySource) != 1 || metrics.BySource[0].Source != "unknown" || metrics.BySource[0].Failed != 1 || metrics.BySource[0].ProbePasses != 1 || metrics.BySource[0].ProbeFailures != 1 {
		t.Fatalf("by source = %#v, want unknown source probe counts", metrics.BySource)
	}
}
