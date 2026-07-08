package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/spacehz-lab/cal/internal/contract"
	"github.com/spacehz-lab/cal/internal/model"
)

func TestCLIShasumCanaryPromotesStdoutHashVerify(t *testing.T) {
	providerPath, err := exec.LookPath("shasum")
	if err != nil {
		t.Skipf("shasum not available: %v", err)
	}

	repo, calctlBin, caldBin := cliCanaryBinariesForTest(t)
	workspace := newCliCanaryWorkspace(t)
	env := cliCanaryEnv(t, workspace.home)
	startCald(t, repo, env, caldBin)

	provider := addProvider(t, repo, env, calctlBin, providerPath)
	acquisition := runAcquisition(t, repo, env, calctlBin, provider.ID, "compute a sha1 checksum for a file and return the hash on stdout")
	trace := assertAcquisitionCompleted(t, acquisition)
	capabilityID := assertCanaryProbe(t, trace, model.VerifyLevelL1, model.VerifyMethodExecute)

	source := filepath.Join(workspace.temp, "checksum-source.txt")
	if err := os.WriteFile(source, []byte("cal checksum canary\n"), 0o644); err != nil {
		t.Fatalf("write checksum source: %v", err)
	}
	var runResponse contract.RunResponse
	runJSON(t, repo, env, &runResponse, calctlBin, "runs", "create", "--capability-id", capabilityID, "--provider-id", provider.ID, "--inputs-json", jsonInputs(t, map[string]any{
		"source":    source,
		"input":     source,
		"target":    filepath.Join(workspace.temp, "checksum.out"),
		"algorithm": "sha1",
	}), "--verify", "--json")
	run := assertRunSucceeded(t, runResponse, capabilityID, provider.ID)
	if stdout, _ := run.Outputs["stdout"].(string); stdout == "" {
		t.Fatalf("shasum outputs = %#v, want stdout hash output", run.Outputs)
	}
}
