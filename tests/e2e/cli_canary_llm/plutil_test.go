package e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
	e2etest "github.com/spacehz-lab/cal/internal/testsupport/e2e"
)

func TestCLIPlutilCanaryPromotesJSONConversion(t *testing.T) {
	providerPath, err := exec.LookPath("plutil")
	if err != nil {
		t.Skipf("plutil not available: %v", err)
	}

	harness := newCliCanaryHarness(t)
	provider, _, trace := harness.discover(t, providerPath, "document.convert")
	probe := assertCliCanaryProbe(t, trace, "document.convert", core.VerifyLevelL2, core.VerifyMethodExecute)
	assertCliCanaryCheck(t, probe, core.VerifySubjectFile, core.VerifyPredicateFormat)

	source := filepath.Join(harness.temp, "source.plist")
	target := filepath.Join(harness.temp, "target.json")
	if err := os.WriteFile(source, []byte(e2etest.PlistFixture("cal-plutil-canary")), 0o644); err != nil {
		t.Fatalf("write plist source: %v", err)
	}
	run := harness.run(t, provider.ID, "document.convert", map[string]any{
		"source": source,
		"target": target,
		"input":  source,
		"output": target,
		"format": "json",
	}, true)
	if run.Status != "succeeded" || !run.Verified || len(run.Evidence) == 0 {
		t.Fatalf("plutil run = %#v, want verified plist to JSON conversion", run)
	}
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read plutil target: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(content, &decoded); err != nil {
		t.Fatalf("decode plutil target JSON: %v\n%s", err, content)
	}
	if decoded["name"] != "cal-plutil-canary" {
		t.Fatalf("plutil target = %#v, want converted plist name", decoded)
	}
}
