package probe

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spacehz-lab/cal/internal/proposal"
)

func TestMaterializeResolvesWorkdirAndWritesFixtures(t *testing.T) {
	workDir := t.TempDir()
	plan, err := Materialize(&Target{
		CandidateIndex: 1,
		WorkDir:        workDir,
		Plan: &proposal.ProbePlan{
			Inputs: map[string]any{
				"target": "{{workdir}}/out.txt",
				"count":  3,
			},
			Fixtures: []proposal.Fixture{{
				Input:    "source",
				Filename: "nested/input.txt",
				Content:  "hello\n",
			}},
		},
	})
	if err != nil {
		t.Fatalf("Materialize() error = %v", err)
	}
	if plan.Inputs["target"] != filepath.Join(workDir, "out.txt") || plan.Inputs["count"] != 3 {
		t.Fatalf("inputs = %#v, want materialized target and scalar count", plan.Inputs)
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

func TestMaterializeRejectsUnresolvedTemplates(t *testing.T) {
	_, err := Materialize(&Target{
		WorkDir: t.TempDir(),
		Plan: &proposal.ProbePlan{
			Inputs: map[string]any{"target": "{{missing}}/out.txt"},
		},
	})
	if err == nil {
		t.Fatal("Materialize() error = nil, want error")
	}
}

func TestMaterializeRejectsEscapingFixture(t *testing.T) {
	_, err := Materialize(&Target{
		WorkDir: t.TempDir(),
		Plan: &proposal.ProbePlan{
			Fixtures: []proposal.Fixture{{Input: "source", Filename: "../secret.txt"}},
		},
	})
	if err == nil {
		t.Fatal("Materialize() error = nil, want error")
	}
}
