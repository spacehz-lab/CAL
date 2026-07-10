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

func TestTextTransformMatches(t *testing.T) {
	source := writeTempFile(t, "source.txt", "hello\nCal\n")
	target := writeTempFile(t, "target.txt", "HELLO\nCAL\n")
	check := fileCheck(model.VerifyPredicateTextTransformMatches, map[string]any{paramSource: "source", paramTransform: transformUppercase})

	if err := runOneCheck(check, map[string]any{"source": source, "target": target}, "", "", 0); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestLineCountMatches(t *testing.T) {
	source := writeTempFile(t, "source.txt", "a\nb\nc\n")
	check := stdoutCheck(model.VerifyPredicateLineCountMatches, map[string]any{paramSource: "source"})

	if err := runOneCheck(check, map[string]any{"source": source}, "       3 source.txt\n", "", 0); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestTextFilterMatches(t *testing.T) {
	source := writeTempFile(t, "source.txt", "info ready\nerror disk\nerror net\n")
	target := writeTempFile(t, "target.txt", "error disk\nerror net\n")
	check := fileCheck(model.VerifyPredicateTextFilterMatches, map[string]any{paramSource: "source", paramPattern: "pattern"})

	if err := runOneCheck(check, map[string]any{"source": source, "target": target, "pattern": "error"}, "", "", 0); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestDelimitedColumnMatches(t *testing.T) {
	source := writeTempFile(t, "source.csv", "name,score\ncal,9\nagent,7\n")
	target := writeTempFile(t, "target.txt", "score\n9\n7\n")
	check := fileCheck(model.VerifyPredicateDelimitedColumnMatch, map[string]any{paramSource: "source", paramDelimiter: "delimiter", paramColumn: "column"})

	if err := runOneCheck(check, map[string]any{"source": source, "target": target, "delimiter": ",", "column": "2"}, "", "", 0); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestDelimitedColumnMatchesHeaderName(t *testing.T) {
	source := writeTempFile(t, "source.csv", "id,name,email\n1,alice,alice@example.com\n2,bob,bob@example.com\n")
	target := writeTempFile(t, "target.txt", "alice@example.com\nbob@example.com\n")
	check := fileCheck(model.VerifyPredicateDelimitedColumnMatch, map[string]any{paramSource: "source", paramDelimiter: ",", paramColumn: "email"})

	if err := runOneCheck(check, map[string]any{"source": source, "target": target}, "", "", 0); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestDelimitedColumnMatchesHeaderNameMissing(t *testing.T) {
	source := writeTempFile(t, "source.csv", "id,name\n1,alice\n")
	target := writeTempFile(t, "target.txt", "alice@example.com\n")
	check := fileCheck(model.VerifyPredicateDelimitedColumnMatch, map[string]any{paramSource: "source", paramDelimiter: ",", paramColumn: "email"})

	if err := runOneCheck(check, map[string]any{"source": source, "target": target}, "", "", 0); err == nil {
		t.Fatal("Run() error = nil, want missing header error")
	}
}

func stdoutCheck(predicate model.VerifyPredicate, params map[string]any) model.VerifyCheck {
	return model.VerifyCheck{
		Subject:   model.VerifySubject{Type: model.VerifySubjectStdout},
		Predicate: predicate,
		Params:    params,
	}
}
