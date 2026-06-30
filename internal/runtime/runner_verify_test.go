package runtime

import (
	"context"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

func TestRunnerVerifyUsesExecutionResultContext(t *testing.T) {
	spec := core.VerifySpec{
		Level:  core.VerifyLevelL2,
		Method: core.VerifyMethodExecute,
		Checks: []core.VerifyCheck{{
			Subject:   core.VerifySubject{Type: core.VerifySubjectStdout},
			Predicate: core.VerifyPredicateContains,
			Params:    map[string]any{"value": "ready"},
		}},
	}

	evidence, outputs, err := NewRunner(DefaultRegistry()).Verify(context.Background(), spec, nil, ExecutionResult{Stdout: "system ready\n"})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if len(evidence) != 1 {
		t.Fatalf("evidence = %#v, want one item", evidence)
	}
	if outputs["stdout"] != "system ready\n" {
		t.Fatalf("outputs = %#v, want stdout output", outputs)
	}
}
