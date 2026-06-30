package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
	calstore "github.com/spacehz-lab/cal/internal/store"
)

func TestReadRunInputsFromJSONAndFile(t *testing.T) {
	inputs, err := readRunInputs(`{"source":"a","target":"b"}`, "")
	if err != nil {
		t.Fatalf("readRunInputs(json) error = %v", err)
	}
	if inputs["source"] != "a" || inputs["target"] != "b" {
		t.Fatalf("inputs = %#v, want source/target", inputs)
	}

	path := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(path, []byte(`{"source":"file"}`), 0o644); err != nil {
		t.Fatalf("write input file: %v", err)
	}
	inputs, err = readRunInputs("", path)
	if err != nil {
		t.Fatalf("readRunInputs(file) error = %v", err)
	}
	if inputs["source"] != "file" {
		t.Fatalf("inputs = %#v, want file source", inputs)
	}

	for _, args := range [][2]string{{"", ""}, {`{"x":1}`, path}, {"[]", ""}, {"{", ""}} {
		if _, err := readRunInputs(args[0], args[1]); err == nil {
			t.Fatalf("readRunInputs(%q, %q) error = nil, want rejection", args[0], args[1])
		}
	}
}

func TestReadOptionalRunInputsAllowsEmpty(t *testing.T) {
	inputs, err := readOptionalRunInputs("", "")
	if err != nil {
		t.Fatalf("readOptionalRunInputs() error = %v", err)
	}
	if len(inputs) != 0 {
		t.Fatalf("inputs = %#v, want empty object", inputs)
	}
}

func TestRunCommandCallsCald(t *testing.T) {
	home := t.TempDir()
	startCLITestCald(t, home)
	providerPath := writeRunScript(t, false)
	seedRunnableCapability(t, home, providerPath, core.ExecutionKindCLI)
	source := filepath.Join(t.TempDir(), "source.txt")
	target := filepath.Join(t.TempDir(), "target.pdf")
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	output, err := executeRoot(home, "runs", "create", "--capability-id", "document.convert", "--binding-id", "binding_cli", "--inputs-json", `{"source":`+strconvQuote(source)+`,"target":`+strconvQuote(target)+`}`, "--verify", "--json")
	if err != nil {
		t.Fatalf("run command error = %v\n%s", err, output)
	}
	var run core.Run
	if err := json.Unmarshal([]byte(output), &run); err != nil {
		t.Fatalf("decode run: %v\n%s", err, output)
	}
	if run.Status != core.RunStatusSucceeded || !run.Verified || run.BindingID != "binding_cli" || run.ProviderID != "provider_cli" {
		t.Fatalf("run = %#v, want verified cald run", run)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("target missing: %v", err)
	}
}

func TestRunCommandReportsCaldUnavailable(t *testing.T) {
	output, err := executeRoot(t.TempDir(), "runs", "create", "--capability-id", "document.convert", "--inputs-json", `{"source":"a","target":"b"}`, "--json")
	if err == nil {
		t.Fatalf("run command succeeded, want cald_unavailable\n%s", output)
	}
	if !strings.Contains(output, `"code": "cald_unavailable"`) {
		t.Fatalf("output = %q, want cald_unavailable", output)
	}
}

func seedRunnableCapability(t *testing.T, home, providerPath string, kind core.ExecutionKind) {
	t.Helper()
	store, err := calstore.Open(home)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := store.PutProvider(core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI, Path: providerPath}); err != nil {
		t.Fatalf("PutProvider() error = %v", err)
	}
	seedCapabilityOnly(t, home, "provider_cli", kind)
}

func seedCapabilityOnly(t *testing.T, home, providerID string, kind core.ExecutionKind) {
	t.Helper()
	store, err := calstore.Open(home)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	capability := core.Capability{
		ID:          "document.convert",
		Description: "Export a document to a PDF artifact.",
		Bindings: []core.Binding{{
			ID:           "binding_cli",
			CapabilityID: "document.convert",
			ProviderID:   providerID,
			Execution: core.Execution{
				Kind: kind,
				Spec: map[string]any{"args": []string{"export-pdf", "--source", "{{source}}", "--target", "{{target}}"}},
			},
			Verify: &core.VerifySpec{
				Level:  core.VerifyLevelL2,
				Method: core.VerifyMethodExecute,
				Checks: []core.VerifyCheck{{Subject: core.VerifySubject{Type: core.VerifySubjectFile, Input: "target"}, Predicate: core.VerifyPredicateExists}},
			},
			Evidence: []core.EvidenceRef{{
				ID: "evidence_file_exists",
			}},
			State: core.BindingStatePromoted,
		}},
	}
	if err := store.PutCapability(capability); err != nil {
		t.Fatalf("PutCapability() error = %v", err)
	}
}

func writeRunScript(t *testing.T, fail bool) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "run-cli")
	body := `#!/bin/sh
if [ "$1" = "export-pdf" ]; then
  if [ "` + boolExit(fail) + `" = "fail" ]; then
    exit 7
  fi
  target=""
  while [ "$#" -gt 0 ]; do
    if [ "$1" = "--target" ]; then
      target="$2"
      break
    fi
    shift
  done
  printf "ok" > "$target"
  exit 0
fi
exit 64
`
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write run script: %v", err)
	}
	return path
}

func boolExit(fail bool) string {
	if fail {
		return "fail"
	}
	return "pass"
}

func strconvQuote(value string) string {
	bytes, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}
