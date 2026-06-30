package proposal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

func TestMaterializeProbePlanResolvesWorkdirAndFixtures(t *testing.T) {
	workDir := t.TempDir()
	plan, err := MaterializeProbePlan(workDir, ProbePlan{
		Inputs: map[string]any{
			"target": "{{workdir}}/output.pdf",
			"count":  3,
		},
		Fixtures: []Fixture{{
			Input:    "source",
			Filename: "input.txt",
			Content:  "hello\n",
		}},
		Verify: core.VerifySpec{
			Level:  core.VerifyLevelL2,
			Method: core.VerifyMethodExecute,
			Checks: []core.VerifyCheck{{Subject: core.VerifySubject{Type: core.VerifySubjectFile, Input: "target"}, Predicate: core.VerifyPredicateExists}},
		},
	})
	if err != nil {
		t.Fatalf("MaterializeProbePlan() error = %v", err)
	}
	if plan.Inputs["target"] != filepath.Join(workDir, "output.pdf") || plan.Inputs["count"] != 3 {
		t.Fatalf("inputs = %#v, want materialized target and retained scalar", plan.Inputs)
	}
	source, ok := plan.Inputs["source"].(string)
	if !ok {
		t.Fatalf("source input = %#v, want string", plan.Inputs["source"])
	}
	content, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("ReadFile(source) error = %v", err)
	}
	if string(content) != "hello\n" {
		t.Fatalf("fixture content = %q, want hello", content)
	}
}

func TestMaterializeProbePlanRejectsUnresolvedTemplate(t *testing.T) {
	_, err := MaterializeProbePlan(t.TempDir(), ProbePlan{
		Inputs: map[string]any{"target": "{{missing}}/output.pdf"},
	})
	if err == nil {
		t.Fatal("MaterializeProbePlan() error = nil, want unresolved template error")
	}
}
