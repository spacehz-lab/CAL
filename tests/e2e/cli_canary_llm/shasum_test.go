package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

func TestCLIShasumCanaryPromotesStdoutHashVerify(t *testing.T) {
	providerPath, err := exec.LookPath("shasum")
	if err != nil {
		t.Skipf("shasum not available: %v", err)
	}

	harness := newCliCanaryHarness(t)
	provider, _, trace := harness.discover(t, providerPath, "file.checksum")
	probe := assertCliCanaryProbe(t, trace, "file.checksum", core.VerifyLevelL2, core.VerifyMethodExecute)
	assertCliCanaryCheck(t, probe, core.VerifySubjectStdout, core.VerifyPredicateHashLineMatches)

	source := filepath.Join(harness.temp, "checksum-source.txt")
	if err := os.WriteFile(source, []byte("cal checksum canary\n"), 0o644); err != nil {
		t.Fatalf("write checksum source: %v", err)
	}
	run := harness.run(t, provider.ID, "file.checksum", map[string]any{
		"source":    source,
		"target":    filepath.Join(harness.temp, "checksum.out"),
		"algorithm": "sha1",
	}, true)
	if run.Status != "succeeded" || !run.Verified || len(run.Evidence) == 0 {
		t.Fatalf("shasum run = %#v, want verified stdout hash run", run)
	}
}
