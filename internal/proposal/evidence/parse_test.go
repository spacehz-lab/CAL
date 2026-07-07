package evidence

import (
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestParseDefaultsLevelAndMethod(t *testing.T) {
	raw := `{"verify":{"checks":[{"subject":{"type":"stdout"},"predicate":"non_empty"}]}}`

	verify, stage, err := Parse(raw, &Request{})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if verify.Level != model.VerifyLevelL1 || verify.Method != model.VerifyMethodExecute {
		t.Fatalf("verify defaults = %q/%q, want L1/execute", verify.Level, verify.Method)
	}
	if stage.Summary[model.ProposalSummaryKeep] != 1 {
		t.Fatalf("keep summary = %d, want 1", stage.Summary[model.ProposalSummaryKeep])
	}
}

func TestParseSkipsUnknownFileInput(t *testing.T) {
	raw := `{"verify":{"checks":[{"subject":{"type":"file","input":"missing"},"predicate":"exists"}]}}`

	_, _, err := Parse(raw, &Request{Material: Material{Inputs: map[string]any{"known": "value"}}})
	if err == nil {
		t.Fatal("Parse() error = nil, want error")
	}
}

func TestParseRejectsExecuteWithoutChecks(t *testing.T) {
	raw := `{"verify":{"method":"execute","checks":[]}}`

	_, _, err := Parse(raw, &Request{})
	if err == nil {
		t.Fatal("Parse() error = nil, want error")
	}
}

func TestParseAllowsContractWithoutChecks(t *testing.T) {
	raw := `{"verify":{"method":"contract","checks":[]}}`

	verify, stage, err := Parse(raw, &Request{})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if verify.Method != model.VerifyMethodContract || verify.Level != model.VerifyLevelL1 || len(verify.Checks) != 0 {
		t.Fatalf("verify = %#v, want L1 contract without checks", verify)
	}
	if stage.Summary[model.ProposalSummarySelected] != 0 {
		t.Fatalf("stage = %#v, want no selected checks", stage)
	}
}

func TestParseSkipsCheckWithMissingPredicateParam(t *testing.T) {
	raw := `{"verify":{"checks":[
		{"subject":{"type":"file","input":"target"},"predicate":"format","params":{}},
		{"subject":{"type":"file","input":"target"},"predicate":"exists"}
	]}}`

	verify, stage, err := Parse(raw, &Request{Material: Material{Inputs: map[string]any{"target": "{{workdir}}/out.pdf"}}})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(verify.Checks) != 1 || verify.Checks[0].Predicate != model.VerifyPredicateExists {
		t.Fatalf("checks = %#v, want only exists", verify.Checks)
	}
	if stage.Summary[model.ProposalSummarySkip] != 1 || stage.Items[0].Reason != "missing predicate param" {
		t.Fatalf("stage = %#v, want missing predicate param skip", stage)
	}
}

func TestParseKeepsCheckWithValidPredicateParam(t *testing.T) {
	raw := `{"verify":{"checks":[{"subject":{"type":"file","input":"target"},"predicate":"format","params":{"format":"json"}}]}}`

	verify, _, err := Parse(raw, &Request{Material: Material{Inputs: map[string]any{"target": "{{workdir}}/out.json"}}})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(verify.Checks) != 1 || verify.Checks[0].Params[paramFormat] != formatJSON {
		t.Fatalf("checks = %#v, want json format check", verify.Checks)
	}
}
