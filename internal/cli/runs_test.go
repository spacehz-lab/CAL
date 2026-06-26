package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/runtime"
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

	output, err := executeRoot(home, "runs", "create", "--capability-id", "document.export_pdf", "--binding-id", "binding_cli", "--inputs-json", `{"source":`+strconvQuote(source)+`,"target":`+strconvQuote(target)+`}`, "--verify", "--json")
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
	output, err := executeRoot(t.TempDir(), "runs", "create", "--capability-id", "document.export_pdf", "--inputs-json", `{"source":"a","target":"b"}`, "--json")
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
	installCLITestVerifier(t, home, "file_exists", fileExistsVerifierScript())
	store, err := calstore.Open(home)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	capability := core.Capability{
		ID:          "document.export_pdf",
		Description: "Export a document to a PDF artifact.",
		Bindings: []core.Binding{{
			ID:           "binding_cli",
			CapabilityID: "document.export_pdf",
			ProviderID:   providerID,
			Execution: core.Execution{
				Kind: kind,
				Spec: map[string]any{"args": []string{"export-pdf", "--source", "{{source}}", "--target", "{{target}}"}},
			},
			Verifier: &core.Verifier{ID: "file_exists"},
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

func installCLITestVerifier(t *testing.T, home, id, verifyPY string) {
	t.Helper()
	t.Setenv("CAL_HOME", home)
	if err := runtime.InstallVerifier(runtime.GeneratedVerifierPackage{
		ID:          id,
		Description: "test verifier",
		VerifyPY:    verifyPY,
	}); err != nil {
		t.Fatalf("InstallVerifier(%s) error = %v", id, err)
	}
}

func fileExistsVerifierScript() string {
	return `import json
import os
import sys

def fail(code, message):
    print(json.dumps({"passed": False, "error": {"code": code, "message": message}}))
    sys.exit(0)

request = json.load(sys.stdin)
verifier_id = request.get("verifier", {}).get("id", "file_exists")
inputs = request.get("inputs") or {}
path = inputs.get("target")
if not isinstance(path, str) or not path:
    fail("missing_input", 'verifier input "target" is required')
if not os.path.exists(path):
    fail("file_missing", f'expected file "{path}" does not exist')
if os.path.isdir(path):
    fail("file_is_directory", f'expected file "{path}" is a directory')
size = os.path.getsize(path)
print(json.dumps({
    "passed": True,
    "evidence": [{
        "id": verifier_id,
        "type": verifier_id,
        "content": {"path": path, "size": size},
    }],
    "outputs": {"target": path},
}))
`
}

func pdfMagicVerifierScript() string {
	return `import json
import os
import sys

def fail(code, message):
    print(json.dumps({"passed": False, "error": {"code": code, "message": message}}))
    sys.exit(0)

request = json.load(sys.stdin)
verifier_id = request.get("verifier", {}).get("id", "pdf_magic_check")
inputs = request.get("inputs") or {}
path = inputs.get("target")
if not isinstance(path, str) or not path:
    fail("missing_input", 'verifier input "target" is required')
if not os.path.exists(path):
    fail("file_missing", f'expected artifact "{path}" does not exist')
if os.path.isdir(path):
    fail("file_is_directory", f'expected artifact "{path}" is a directory')
with open(path, "rb") as handle:
    content = handle.read().strip()
ok = (
    content.startswith(b"%PDF-")
    and b"xref" in content
    and b"trailer" in content
    and b"startxref" in content
    and content.endswith(b"%%EOF")
)
if not ok:
    fail("parse_failed", f'artifact "{path}" did not pass pdf parse checks')
print(json.dumps({
    "passed": True,
    "evidence": [{
        "id": verifier_id,
        "type": verifier_id,
        "content": {"path": path, "size": os.path.getsize(path), "format": "pdf"},
    }],
    "outputs": {"target": path},
}))
`
}
