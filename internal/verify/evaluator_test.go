package verify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

func TestEvaluateTextPredicatesReadPathContent(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "report.json")
	if err := os.WriteFile(target, []byte(`{"status":"ok","checks":[{"result":"ok"}]}`), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	spec := core.VerifySpec{
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

	evidence, outputs, err := Evaluate(context.Background(), spec, Context{Inputs: map[string]any{"target": target}})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if len(evidence) != 3 {
		t.Fatalf("evidence = %#v, want three items", evidence)
	}
	if outputs["target"] != target {
		t.Fatalf("outputs = %#v, want target path", outputs)
	}
}

func TestEvaluateTextPredicateReportsMissingPath(t *testing.T) {
	spec := core.VerifySpec{
		Level:  core.VerifyLevelL2,
		Method: core.VerifyMethodExecute,
		Checks: []core.VerifyCheck{{
			Subject:   core.VerifySubject{Type: core.VerifySubjectFile, Input: "target"},
			Predicate: core.VerifyPredicateContains,
			Params:    map[string]any{"value": "ok"},
		}},
	}

	_, _, err := Evaluate(context.Background(), spec, Context{Inputs: map[string]any{"target": filepath.Join(t.TempDir(), "missing.txt")}})
	if err == nil || !strings.Contains(err.Error(), "verify subject read") {
		t.Fatalf("Evaluate() error = %v, want missing path read error", err)
	}
}

func TestEvaluateContainsUsesStdoutValue(t *testing.T) {
	spec := core.VerifySpec{
		Level:  core.VerifyLevelL2,
		Method: core.VerifyMethodExecute,
		Checks: []core.VerifyCheck{{
			Subject:   core.VerifySubject{Type: core.VerifySubjectStdout},
			Predicate: core.VerifyPredicateContains,
			Params:    map[string]any{"value": "ready"},
		}},
	}

	evidence, outputs, err := Evaluate(context.Background(), spec, Context{Stdout: "system ready\n"})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if len(evidence) != 1 {
		t.Fatalf("evidence = %#v, want one item", evidence)
	}
	if outputs["stdout"] != "system ready\n" {
		t.Fatalf("outputs = %#v, want stdout output", outputs)
	}
}

func TestEvaluateHashLineMatchesNormalizesAlgorithmName(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.txt")
	if err := os.WriteFile(source, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	sum := sha256.Sum256([]byte("hello\n"))
	stdout := hex.EncodeToString(sum[:]) + "  source.txt\n"
	spec := core.VerifySpec{
		Level:  core.VerifyLevelL3,
		Method: core.VerifyMethodExecute,
		Checks: []core.VerifyCheck{{
			Subject:   core.VerifySubject{Type: core.VerifySubjectStdout},
			Predicate: core.VerifyPredicateHashLineMatches,
			Params:    map[string]any{"source": "source", "algorithm": "SHA-256"},
		}},
	}

	if _, _, err := Evaluate(context.Background(), spec, Context{Inputs: map[string]any{"source": source}, Stdout: stdout}); err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
}

func TestVerifySubjectRulesHavePredicateHandlers(t *testing.T) {
	for _, rule := range core.VerifySubjectRules() {
		for _, predicate := range rule.AllowedPredicates {
			if !hasPredicateHandler(predicate) {
				t.Fatalf("subject %s allows predicate %s without verify handler", rule.Type, predicate)
			}
		}
	}
}
