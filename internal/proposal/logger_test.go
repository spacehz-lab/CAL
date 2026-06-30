package proposal

import (
	"testing"

	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestBindingStageLogAttrsSummarizesRejectedCandidates(t *testing.T) {
	attrs := bindingStageLogAttrs([]caltrace.ProposalStage{{
		Name: caltrace.ProposalStageBinding,
		Summary: map[caltrace.ProposalSummaryKey]int{
			caltrace.ProposalSummaryRaw:      2,
			caltrace.ProposalSummarySkip:     1,
			caltrace.ProposalSummaryDefer:    1,
			caltrace.ProposalSummarySelected: 0,
		},
		Items: []caltrace.ProposalItem{
			{Decision: caltrace.ProposalDecisionSkip, Reason: bindingReasonInvalidCLIArgsType},
			{Decision: caltrace.ProposalDecisionDefer, Reason: bindingReasonCandidateLimit},
		},
	}})

	values := attrMap(attrs)
	if values[logKeyRawCandidateCount] != 2 || values[logKeySelectedCandidateCount] != 0 || values[logKeySkipCount] != 1 || values[logKeyDeferCount] != 1 {
		t.Fatalf("attrs = %#v, want binding summary counts", values)
	}
	if got := values[logKeySkipReasons].([]string); len(got) != 1 || got[0] != bindingReasonInvalidCLIArgsType {
		t.Fatalf("skip reasons = %#v, want invalid cli args type", got)
	}
	if got := values[logKeyDeferReasons].([]string); len(got) != 1 || got[0] != bindingReasonCandidateLimit {
		t.Fatalf("defer reasons = %#v, want candidate limit", got)
	}
}

func attrMap(attrs []any) map[string]any {
	values := map[string]any{}
	for index := 0; index+1 < len(attrs); index += 2 {
		key, ok := attrs[index].(string)
		if !ok {
			continue
		}
		values[key] = attrs[index+1]
	}
	return values
}
