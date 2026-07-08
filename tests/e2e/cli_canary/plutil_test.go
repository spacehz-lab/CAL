package e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spacehz-lab/cal/internal/contract"
	"github.com/spacehz-lab/cal/internal/model"
)

func TestCLIPlutilCanaryPromotesJSONConversion(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("plutil canary requires macOS")
	}
	providerPath, err := exec.LookPath("plutil")
	if err != nil {
		t.Skipf("plutil not available: %v", err)
	}

	repo, calctlBin, caldBin := cliCanaryBinariesForTest(t)
	workspace := newCliCanaryWorkspace(t)
	env := cliCanaryEnv(t, workspace.home)
	startCald(t, repo, env, caldBin)

	provider := addProvider(t, repo, env, calctlBin, providerPath)
	acquisition := runAcquisition(t, repo, env, calctlBin, provider.ID, "convert a property list file to json")
	trace := assertAcquisitionCompleted(t, acquisition)
	capabilityID := assertCanaryProbe(t, trace, model.VerifyLevelL2, model.VerifyMethodExecute,
		check(model.VerifySubjectFile, model.VerifyPredicateFormat),
	)

	source := filepath.Join(workspace.temp, "source.plist")
	target := filepath.Join(workspace.temp, "target.json")
	if err := os.WriteFile(source, []byte(plistFixture("cal-plutil-canary")), 0o644); err != nil {
		t.Fatalf("write plist source: %v", err)
	}
	var runResponse contract.RunResponse
	runJSON(t, repo, env, &runResponse, calctlBin, "runs", "create", "--capability-id", capabilityID, "--provider-id", provider.ID, "--inputs-json", jsonInputs(t, map[string]any{
		"source": source,
		"target": target,
		"input":  source,
		"output": target,
		"format": "json",
	}), "--verify", "--json")
	assertRunSucceeded(t, runResponse, capabilityID, provider.ID)
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

func plistFixture(name string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict><key>name</key><string>` + name + `</string></dict></plist>
`
}
