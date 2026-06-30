package e2e

import (
	"os/exec"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

func TestCLIBrewCanaryPromotesContractAndSafeExecute(t *testing.T) {
	providerPath, err := exec.LookPath("brew")
	if err != nil {
		t.Skipf("brew not available: %v", err)
	}

	harness := newCliCanaryHarness(t)
	packageProvider, _, packageTrace := harness.discover(t, providerPath, "package.install")
	assertCliCanaryProbe(t, packageTrace, "package.install", core.VerifyLevelL1, core.VerifyMethodContract)
	assertCliCanaryCandidateArg(t, packageTrace, "package.install", "install")

	packageRun := harness.runFail(t, packageProvider.ID, "package.install", map[string]any{
		"package": "cal-contract-only-example",
	})
	if packageRun.Status != "failed" || packageRun.Error == nil || packageRun.Error.Code != "binding_not_found" {
		t.Fatalf("package.install default run = %#v, want contract L1 filtered below default L2 threshold", packageRun)
	}

	executeProvider, _, executeTrace := harness.discover(t, providerPath, "system.inspect")
	assertCliCanaryProbe(t, executeTrace, "system.inspect", core.VerifyLevelL2, core.VerifyMethodExecute)
	assertCliCanaryCandidateArg(t, executeTrace, "system.inspect", "config")

	inspectRun := harness.run(t, executeProvider.ID, "system.inspect", map[string]any{}, true)
	if inspectRun.Status != "succeeded" || !inspectRun.Verified || len(inspectRun.Evidence) == 0 {
		t.Fatalf("system.inspect run = %#v, want verified safe brew command execution", inspectRun)
	}
	if output, _ := inspectRun.Outputs["stdout"].(string); output == "" {
		if output, _ := inspectRun.Outputs["output"].(string); output == "" {
			t.Fatalf("system.inspect outputs = %#v, want captured brew output", inspectRun.Outputs)
		}
	}
}
