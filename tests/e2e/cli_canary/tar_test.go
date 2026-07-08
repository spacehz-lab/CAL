package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/spacehz-lab/cal/internal/contract"
	"github.com/spacehz-lab/cal/internal/model"
)

func TestCLITarCanaryPromotesArchiveCompression(t *testing.T) {
	providerPath, err := exec.LookPath("tar")
	if err != nil {
		t.Skipf("tar not available: %v", err)
	}

	repo, calctlBin, caldBin := cliCanaryBinariesForTest(t)
	workspace := newCliCanaryWorkspace(t)
	env := cliCanaryEnv(t, workspace.home)
	startCald(t, repo, env, caldBin)

	provider := addProvider(t, repo, env, calctlBin, providerPath)
	acquisition := runAcquisition(t, repo, env, calctlBin, provider.ID, "compress a file into a tar gzip archive at an output path")
	trace := assertAcquisitionCompleted(t, acquisition)
	capabilityID := assertCanaryProbe(t, trace, model.VerifyLevelL1, model.VerifyMethodExecute)

	source := filepath.Join(workspace.temp, "archive-source.txt")
	target := filepath.Join(workspace.temp, "archive.tar.gz")
	if err := os.WriteFile(source, []byte("cal tar canary\n"), 0o644); err != nil {
		t.Fatalf("write tar source: %v", err)
	}
	var runResponse contract.RunResponse
	runJSON(t, repo, env, &runResponse, calctlBin, "runs", "create", "--capability-id", capabilityID, "--provider-id", provider.ID, "--inputs-json", jsonInputs(t, map[string]any{
		"source":  source,
		"input":   source,
		"file":    source,
		"target":  target,
		"output":  target,
		"archive": target,
	}), "--verify", "--json")
	assertRunSucceeded(t, runResponse, capabilityID, provider.ID)
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat tar target: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("tar target %s is empty", target)
	}
}
