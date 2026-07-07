package e2e

import (
	"os"
	"path/filepath"
	goruntime "runtime"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestCupsfilterRunsRealLocalCLIBinding(t *testing.T) {
	if os.Getenv("CAL_LOCAL_CLI_E2E") != "1" {
		t.Skip("set CAL_LOCAL_CLI_E2E=1 to run local real-CLI e2e")
	}
	if goruntime.GOOS != "darwin" {
		t.Skip("cupsfilter integration requires macOS")
	}
	providerPath := "/usr/sbin/cupsfilter"
	if _, err := os.Stat(providerPath); err != nil {
		t.Skipf("cupsfilter not available: %v", err)
	}

	workspace := newLocalCLIWorkspace(t, providerPath)
	capability := workspace.seedCapability(t, "document.convert", "Export or convert a document or text file into a PDF artifact.", model.Execution{
		Kind: model.ExecutionKindCLI,
		Spec: map[string]any{
			model.ExecutionSpecArgs:            []string{"-i", "text/plain", "-m", "application/pdf", "{{source}}"},
			model.ExecutionSpecStdoutPathInput: "target",
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
				Params:    map[string]any{verifyParamFormat: "pdf"},
			},
		},
	})

	source := filepath.Join(t.TempDir(), "source.txt")
	target := filepath.Join(t.TempDir(), "target.pdf")
	if err := os.WriteFile(source, []byte("CAL cupsfilter integration\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	var direct runResponse
	workspace.runJSON(t, &direct, "runs", "create", "--capability-id", capability.ID, "--provider-id", workspace.provider.ID, "--inputs-json", jsonInputs(map[string]any{"source": source, "target": target}), "--verify", "--json")
	assertSuccessfulVerifiedRun(t, direct.Run, capability.ID, workspace.provider.ID)
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("direct target missing: %v", err)
	}

	useTarget := filepath.Join(t.TempDir(), "use-target.pdf")
	var use useResponse
	workspace.runJSON(t, &use, "use", "export this text document as pdf", "--provider-id", workspace.provider.ID, "--inputs-json", jsonInputs(map[string]any{"source": source, "target": useTarget}), "--verify", "--json")
	if use.Selection == nil || use.Selection.CapabilityID != capability.ID || use.Selection.ProviderID != workspace.provider.ID {
		t.Fatalf("use selection = %#v, want cupsfilter document.convert binding", use.Selection)
	}
	assertSuccessfulVerifiedRun(t, use.Run, capability.ID, workspace.provider.ID)
	if _, err := os.Stat(useTarget); err != nil {
		t.Fatalf("use target missing: %v", err)
	}

	var metrics evalResponse
	workspace.runJSON(t, &metrics, "eval", "--json")
	assertEval(t, metrics, 2, 2)
}
