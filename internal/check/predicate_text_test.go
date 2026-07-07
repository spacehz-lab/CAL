package check

import (
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestTextPredicatesPassAndFail(t *testing.T) {
	tests := []struct {
		name    string
		check   model.VerifyCheck
		stdout  string
		wantErr bool
	}{
		{name: "equals pass", stdout: "ok", check: stdoutCheck(model.VerifyPredicateEquals, map[string]any{paramValue: "ok"})},
		{name: "equals fail", stdout: "bad", check: stdoutCheck(model.VerifyPredicateEquals, map[string]any{paramValue: "ok"}), wantErr: true},
		{name: "not equals pass", stdout: "bad", check: stdoutCheck(model.VerifyPredicateNotEquals, map[string]any{paramValue: "ok"})},
		{name: "not equals fail", stdout: "ok", check: stdoutCheck(model.VerifyPredicateNotEquals, map[string]any{paramValue: "ok"}), wantErr: true},
		{name: "contains pass", stdout: "system ready", check: stdoutCheck(model.VerifyPredicateContains, map[string]any{paramValue: "ready"})},
		{name: "contains fail", stdout: "system down", check: stdoutCheck(model.VerifyPredicateContains, map[string]any{paramValue: "ready"}), wantErr: true},
		{name: "contains any pass", stdout: "status ok", check: stdoutCheck(model.VerifyPredicateContainsAny, map[string]any{paramValues: []any{"warn", "ok"}})},
		{name: "contains any fail", stdout: "status bad", check: stdoutCheck(model.VerifyPredicateContainsAny, map[string]any{paramValues: []any{"warn", "ok"}}), wantErr: true},
		{name: "regex pass", stdout: `{"checks":[]}`, check: stdoutCheck(model.VerifyPredicateRegex, map[string]any{paramPattern: `"checks":\[`})},
		{name: "regex fail", stdout: `{"items":[]}`, check: stdoutCheck(model.VerifyPredicateRegex, map[string]any{paramPattern: `"checks":\[`}), wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := runOneCheck(test.check, nil, test.stdout, "", 0)
			if test.wantErr && err == nil {
				t.Fatal("Run() error = nil, want error")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("Run() error = %v", err)
			}
		})
	}
}

func stdoutCheck(predicate model.VerifyPredicate, params map[string]any) model.VerifyCheck {
	return model.VerifyCheck{
		Subject:   model.VerifySubject{Type: model.VerifySubjectStdout},
		Predicate: predicate,
		Params:    params,
	}
}
