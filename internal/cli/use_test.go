package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
	caluse "github.com/spacehz-lab/cal/internal/use"
)

func TestUseCommandCallsCald(t *testing.T) {
	home := t.TempDir()
	startCLITestCald(t, home)
	providerPath := writeRunScript(t, false)
	seedRunnableCapability(t, home, providerPath, core.ExecutionKindCLI)
	source := filepath.Join(t.TempDir(), "source.txt")
	target := filepath.Join(t.TempDir(), "target.pdf")
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	output, err := executeRoot(home, "use", "--intent", "export this document as pdf", "--inputs-json", `{"source":`+strconvQuote(source)+`,"target":`+strconvQuote(target)+`}`, "--verify", "--json")
	if err != nil {
		t.Fatalf("use command error = %v\n%s", err, output)
	}
	var result caluse.Result
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode use result: %v\n%s", err, output)
	}
	if result.Status != core.RunStatusSucceeded || result.Selection == nil || result.Run == nil || !result.Run.Verified {
		t.Fatalf("use result = %#v, want verified successful use", result)
	}
	if result.Selection.CapabilityID != "document.convert" || result.Selection.BindingID != "binding_cli" || result.Run.BindingID != "binding_cli" {
		t.Fatalf("use selection/run = %#v / %#v, want binding_cli", result.Selection, result.Run)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("target missing: %v", err)
	}
}

func TestUseCommandReportsCaldUnavailable(t *testing.T) {
	output, err := executeRoot(t.TempDir(), "use", "export pdf", "--json")
	if err == nil {
		t.Fatalf("use command succeeded, want cald_unavailable\n%s", output)
	}
	if !strings.Contains(output, `"code": "cald_unavailable"`) {
		t.Fatalf("output = %q, want cald_unavailable", output)
	}
}
