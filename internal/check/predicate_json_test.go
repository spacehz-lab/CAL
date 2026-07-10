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

func TestJSONQueryMatchesSupportsDollarPrefixedPath(t *testing.T) {
	source := writeTempFile(t, "source.json", `{"items":[{"id":7}]}`)
	check := stdoutCheck(model.VerifyPredicateJSONQueryMatches, map[string]any{paramSource: "source", paramQuery: "$.items.0.id"})

	if err := runOneCheck(check, map[string]any{"source": source}, "7\n", "", 0); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestJSONEquivalentPassesAndFails(t *testing.T) {
	source := writeTempFile(t, "source.json", `{"project":{"name":"cal","version":"0.0.1"}}`)
	target := writeTempFile(t, "target.json", "{\n  \"project\": {\"version\": \"0.0.1\", \"name\": \"cal\"}\n}\n")
	bad := writeTempFile(t, "bad.json", `{"project":{"name":"other","version":"0.0.1"}}`)
	check := fileCheck(model.VerifyPredicateJSONEquivalent, map[string]any{paramSource: "source"})

	if err := runOneCheck(check, map[string]any{"source": source, "target": target}, "", "", 0); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if err := runOneCheck(check, map[string]any{"source": source, "target": bad}, "", "", 0); err == nil {
		t.Fatal("Run() error = nil, want mismatch error")
	}
}

func TestJSONFieldEqualsPassesAndFails(t *testing.T) {
	target := writeTempFile(t, "target.json", `{"label":"release","nested":{"status":"ok"}}`)
	bad := writeTempFile(t, "bad.json", `{"label":"debug"}`)
	check := fileCheck(model.VerifyPredicateJSONFieldEquals, map[string]any{paramQuery: ".label", paramValue: "label"})

	if err := runOneCheck(check, map[string]any{"target": target, "label": "release"}, "", "", 0); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if err := runOneCheck(check, map[string]any{"target": bad, "label": "release"}, "", "", 0); err == nil {
		t.Fatal("Run() error = nil, want mismatch error")
	}
}

func TestJSONFieldMatchesSourcePassesAndFails(t *testing.T) {
	source := writeTempFile(t, "artifact.txt", "hello\n")
	target := writeTempFile(t, "target.json", `{"basename":"artifact.txt","bytes":6,"sha256":"5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03"}`)
	bad := writeTempFile(t, "bad.json", `{"basename":"artifact.txt","bytes":7}`)

	tests := []struct {
		name     string
		query    string
		property string
		target   string
		wantErr  bool
	}{
		{name: "basename", query: ".basename", property: sourcePropertyBasename, target: target},
		{name: "bytes", query: ".bytes", property: sourcePropertyBytes, target: target},
		{name: "sha256", query: ".sha256", property: sourcePropertySHA256, target: target},
		{name: "mismatch", query: ".bytes", property: sourcePropertyBytes, target: bad, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			check := fileCheck(model.VerifyPredicateJSONFieldMatchesSource, map[string]any{paramQuery: test.query, paramSource: "source", paramProperty: test.property})
			err := runOneCheck(check, map[string]any{"source": source, "target": test.target}, "", "", 0)
			if test.wantErr && err == nil {
				t.Fatal("Run() error = nil, want mismatch error")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("Run() error = %v", err)
			}
		})
	}
}
