package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

func TestEvaluateVerifySpecTextPredicatesReadPathContent(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "report.json")
	if err := os.WriteFile(target, []byte(`{"status":"ok","checks":[{"result":"ok"}]}`), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	verify := core.VerifySpec{
		Level:  core.VerifyLevelL2,
		Method: core.VerifyMethodExecute,
		Checks: []core.VerifyCheck{
			{
				Subject:   core.VerifySubject{Type: core.VerifySubjectFile, Input: "target"},
				Predicate: core.VerifyPredicateContains,
				Params:    map[string]any{"value": `"status":"ok"`},
			},
			{
				Subject:   core.VerifySubject{Type: core.VerifySubjectFile, Input: "target"},
				Predicate: core.VerifyPredicateRegex,
				Params:    map[string]any{"pattern": `"checks":\[`},
			},
			{
				Subject:   core.VerifySubject{Type: core.VerifySubjectFile, Input: "target"},
				Predicate: core.VerifyPredicateContainsAny,
				Params:    map[string]any{"values": []any{`"result":"warn"`, `"result":"ok"`}},
			},
		},
	}

	evidence, outputs, err := evaluateVerifySpec(context.Background(), verify, map[string]any{"target": target}, ExecutionResult{})
	if err != nil {
		t.Fatalf("evaluateVerifySpec() error = %v", err)
	}
	if len(evidence) != 3 {
		t.Fatalf("evidence = %#v, want three items", evidence)
	}
	if outputs["target"] != target {
		t.Fatalf("outputs = %#v, want target path", outputs)
	}
}

func TestEvaluateVerifySpecTextPredicateReportsMissingPath(t *testing.T) {
	verify := core.VerifySpec{
		Level:  core.VerifyLevelL2,
		Method: core.VerifyMethodExecute,
		Checks: []core.VerifyCheck{{
			Subject:   core.VerifySubject{Type: core.VerifySubjectFile, Input: "target"},
			Predicate: core.VerifyPredicateContains,
			Params:    map[string]any{"value": "ok"},
		}},
	}

	_, _, err := evaluateVerifySpec(context.Background(), verify, map[string]any{"target": filepath.Join(t.TempDir(), "missing.txt")}, ExecutionResult{})
	if err == nil || !strings.Contains(err.Error(), "verify subject read") {
		t.Fatalf("evaluateVerifySpec() error = %v, want missing path read error", err)
	}
}

func TestEvaluateVerifySpecContainsUsesStdoutValue(t *testing.T) {
	verify := core.VerifySpec{
		Level:  core.VerifyLevelL2,
		Method: core.VerifyMethodExecute,
		Checks: []core.VerifyCheck{{
			Subject:   core.VerifySubject{Type: core.VerifySubjectStdout},
			Predicate: core.VerifyPredicateContains,
			Params:    map[string]any{"value": "ready"},
		}},
	}

	evidence, outputs, err := evaluateVerifySpec(context.Background(), verify, nil, ExecutionResult{Stdout: "system ready\n"})
	if err != nil {
		t.Fatalf("evaluateVerifySpec() error = %v", err)
	}
	if len(evidence) != 1 {
		t.Fatalf("evidence = %#v, want one item", evidence)
	}
	if outputs["stdout"] != "system ready\n" {
		t.Fatalf("outputs = %#v, want stdout output", outputs)
	}
}
