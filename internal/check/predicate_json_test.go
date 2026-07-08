package check

import (
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestJSONQueryMatchesPassesAndFails(t *testing.T) {
	source := writeTempFile(t, "source.json", `{"project":{"name":"cal","version":"0.0.1"},"items":[{"id":7}]}`)
	target := writeTempFile(t, "target.txt", "cal\n")
	bad := writeTempFile(t, "bad.txt", "other\n")
	check := fileCheck(model.VerifyPredicateJSONQueryMatches, map[string]any{paramSource: "source", paramQuery: "query"})

	inputs := map[string]any{"source": source, "query": ".project.name", "target": target}
	if err := runOneCheck(check, inputs, "", "", 0); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	inputs["target"] = bad
	if err := runOneCheck(check, inputs, "", "", 0); err == nil {
		t.Fatal("Run() error = nil, want mismatch error")
	}
}

func TestJSONQueryMatchesSupportsLiteralQueryAndStdout(t *testing.T) {
	source := writeTempFile(t, "source.json", `{"items":[{"id":7}]}`)
	check := stdoutCheck(model.VerifyPredicateJSONQueryMatches, map[string]any{paramSource: "source", paramQuery: ".items.0.id"})

	if err := runOneCheck(check, map[string]any{"source": source}, "7\n", "", 0); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}
