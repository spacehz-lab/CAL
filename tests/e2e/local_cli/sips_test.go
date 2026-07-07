package e2e

import (
	"os"
	"path/filepath"
	goruntime "runtime"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestSipsRunsRealLocalCLIBinding(t *testing.T) {
	if os.Getenv("CAL_LOCAL_CLI_E2E") != "1" {
		t.Skip("set CAL_LOCAL_CLI_E2E=1 to run local real-CLI e2e")
	}
	if goruntime.GOOS != "darwin" {
		t.Skip("sips integration requires macOS")
	}
	providerPath := "/usr/bin/sips"
	if _, err := os.Stat(providerPath); err != nil {
		t.Skipf("sips not available: %v", err)
	}

	workspace := newLocalCLIWorkspace(t, providerPath)
	capability := workspace.seedCapability(t, "image.resize", "Resize an image artifact to requested dimensions.", model.Execution{
		Kind: model.ExecutionKindCLI,
		Spec: map[string]any{
			model.ExecutionSpecArgs: []string{"--resampleHeightWidth", "{{height}}", "{{width}}", "{{source}}", "--out", "{{target}}"},
		},
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
				Subject:   model.VerifySubject{Type: model.VerifySubjectFile, Input: "target"},
				Predicate: model.VerifyPredicateFormat,
				Params:    map[string]any{verifyParamFormat: "png"},
			},
		},
	})

	source := filepath.Join(t.TempDir(), "source.png")
	target := filepath.Join(t.TempDir(), "target.png")
	writePNG(t, source, 5, 5)
	var direct runResponse
	workspace.runJSON(t, &direct, "runs", "create", "--capability-id", capability.ID, "--provider-id", workspace.provider.ID, "--inputs-json", jsonInputs(map[string]any{"source": source, "target": target, "width": 12, "height": 8}), "--verify", "--json")
	assertSuccessfulVerifiedRun(t, direct.Run, capability.ID, workspace.provider.ID)
	assertPNGDimensions(t, target, 12, 8)

	useTarget := filepath.Join(t.TempDir(), "use-target.png")
	var use useResponse
	workspace.runJSON(t, &use, "use", "resize this image", "--provider-id", workspace.provider.ID, "--inputs-json", jsonInputs(map[string]any{"source": source, "target": useTarget, "width": 16, "height": 9}), "--verify", "--json")
	if use.Selection == nil || use.Selection.CapabilityID != capability.ID || use.Selection.ProviderID != workspace.provider.ID {
		t.Fatalf("use selection = %#v, want sips image.resize binding", use.Selection)
	}
	assertSuccessfulVerifiedRun(t, use.Run, capability.ID, workspace.provider.ID)
	assertPNGDimensions(t, useTarget, 16, 9)

	var metrics evalResponse
	workspace.runJSON(t, &metrics, "eval", "--json")
	assertEval(t, metrics, 2, 2)
}
