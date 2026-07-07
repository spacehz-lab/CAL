package e2e

import (
	"os"
	"os/exec"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestBrewRunsRealLocalCLIBinding(t *testing.T) {
	if os.Getenv("CAL_LOCAL_CLI_E2E") != "1" {
		t.Skip("set CAL_LOCAL_CLI_E2E=1 to run local real-CLI e2e")
	}
	providerPath, err := exec.LookPath("brew")
	if err != nil {
		t.Skipf("brew not available: %v", err)
	}

	workspace := newLocalCLIWorkspace(t, providerPath)
	capability := workspace.seedCapability(t, "system.inspect", "Inspect local Homebrew configuration.", model.Execution{
		Kind: model.ExecutionKindCLI,
		Spec: map[string]any{model.ExecutionSpecArgs: []string{"config"}},
	}, &model.VerifySpec{
		Level:  model.VerifyLevelL2,
		Method: model.VerifyMethodExecute,
		Checks: []model.VerifyCheck{
			{
				Subject:   model.VerifySubject{Type: model.VerifySubjectExitCode},
				Predicate: model.VerifyPredicateEquals,
				Params:    map[string]any{verifyParamValue: 0},
			},
			{
				Subject:   model.VerifySubject{Type: model.VerifySubjectStdout},
				Predicate: model.VerifyPredicateContains,
				Params:    map[string]any{verifyParamValue: "HOMEBREW_"},
			},
		},
	})

	var direct runResponse
	workspace.runJSON(t, &direct, "runs", "create", "--capability-id", capability.ID, "--provider-id", workspace.provider.ID, "--inputs-json", `{}`, "--verify", "--json")
	assertSuccessfulVerifiedRun(t, direct.Run, capability.ID, workspace.provider.ID)
	if output, _ := direct.Run.Outputs["stdout"].(string); output == "" {
		t.Fatalf("system.inspect outputs = %#v, want captured brew config stdout", direct.Run.Outputs)
	}

	var metrics evalResponse
	workspace.runJSON(t, &metrics, "eval", "--json")
	assertEval(t, metrics, 1, 1)
}
