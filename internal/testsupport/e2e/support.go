package e2e

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/spacehz-lab/cal/internal/calpath"
	"github.com/spacehz-lab/cal/internal/core"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

// ProviderSummary is the provider shape returned by calctl provider commands.
type ProviderSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
	Path string `json:"path"`
}

// EvalMetricsOutput is the eval JSON shape used by end-to-end tests.
type EvalMetricsOutput struct {
	Summary struct {
		Providers        int `json:"providers"`
		Capabilities     int `json:"capabilities"`
		Bindings         int `json:"bindings"`
		PromotedBindings int `json:"promoted_bindings"`
		Traces           int `json:"traces"`
		Runs             int `json:"runs"`
	} `json:"summary"`
	Acquisition struct {
		AttemptCount           int     `json:"attempt_count"`
		CompletedCount         int     `json:"completed_count"`
		PromotionCount         int     `json:"promotion_count"`
		CapabilityCreatedCount int     `json:"capability_created_count"`
		CapabilityReusedCount  int     `json:"capability_reused_count"`
		BindingCreatedCount    int     `json:"binding_created_count"`
		BindingUpdatedCount    int     `json:"binding_updated_count"`
		CandidateCount         int     `json:"candidate_count"`
		ProbePassCount         int     `json:"probe_pass_count"`
		BindingPromotionRate   float64 `json:"binding_promotion_rate"`
		ProbeSuccessRate       float64 `json:"probe_success_rate"`
	} `json:"acquisition"`
	Reuse struct {
		RunCount            int     `json:"run_count"`
		RunSuccessCount     int     `json:"run_success_count"`
		RunFailureCount     int     `json:"run_failure_count"`
		VerifiedRunCount    int     `json:"verified_run_count"`
		VerifierFailCount   int     `json:"verifier_fail_count"`
		RunSuccessRate      float64 `json:"run_success_rate"`
		VerifiedSuccessRate float64 `json:"verified_success_rate"`
		VerifierFailureRate float64 `json:"verifier_failure_rate"`
	} `json:"reuse"`
}

// RepoRoot returns the repository root from this support package.
func RepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

// Build compiles one Go package into output.
func Build(t *testing.T, repo, output, pkg string) {
	t.Helper()
	cmd := exec.Command("go", "build", "-o", output, pkg)
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build %s failed: %v\n%s", pkg, err, out)
	}
}

// StartCald starts a cald serve process and waits for its HTTP endpoint.
func StartCald(t *testing.T, repo string, env []string, caldBin string) {
	t.Helper()
	cmd := exec.Command(caldBin, "serve")
	cmd.Dir = repo
	cmd.Env = env
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start cald: %v\n%s", err, stderr.String())
	}
	done := make(chan struct{})
	var waitErr error
	go func() {
		waitErr = cmd.Wait()
		close(done)
	}()
	t.Cleanup(func() {
		select {
		case <-done:
		default:
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			<-done
		}
	})
	waitForCaldEndpoint(t, env, &stderr, done, func() error { return waitErr })
}

// RunJSON runs a command and decodes stdout as JSON.
func RunJSON(t *testing.T, repo string, env []string, target any, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = repo
	cmd.Env = env
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("%s %s failed: %v\nstderr=%s\nstdout=%s", name, strings.Join(args, " "), err, stderr.String(), out)
	}
	if err := json.Unmarshal(out, target); err != nil {
		t.Fatalf("%s %s returned invalid JSON: %v\nstderr=%s\nstdout=%s", name, strings.Join(args, " "), err, stderr.String(), out)
	}
}

type caldEndpoint struct {
	BaseURL string `json:"base_url"`
}

func waitForCaldEndpoint(t *testing.T, env []string, stderr *bytes.Buffer, done <-chan struct{}, waitErr func() error) {
	t.Helper()
	home := HomeFromEnv(env)
	if home == "" {
		t.Fatal("CAL home is required to wait for cald endpoint")
	}
	endpointPath := filepath.Join(home, "cald", "endpoint.json")
	client := http.Client{Timeout: time.Second}
	var lastErr error
	started := time.Now()
	deadline := started.Add(10 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-done:
			t.Fatalf("cald exited before endpoint became ready after %s: %v\nendpoint=%s\nstderr=%s", time.Since(started).Round(time.Millisecond), waitErr(), endpointPath, stderr.String())
		default:
		}
		content, err := os.ReadFile(endpointPath)
		if err == nil {
			var endpoint caldEndpoint
			if err := json.Unmarshal(content, &endpoint); err == nil && endpoint.BaseURL != "" {
				resp, err := client.Get(strings.TrimRight(endpoint.BaseURL, "/") + "/v1/daemon/status")
				if err == nil {
					_ = resp.Body.Close()
					if resp.StatusCode == http.StatusOK {
						return
					}
				}
				lastErr = err
			} else {
				lastErr = err
			}
		} else {
			lastErr = err
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("cald endpoint did not become ready after %s: %v\nendpoint=%s\nstderr=%s", time.Since(started).Round(time.Millisecond), lastErr, endpointPath, stderr.String())
}

// RunFailJSON runs a command that is expected to fail and decodes stdout as JSON.
func RunFailJSON(t *testing.T, repo string, env []string, target any, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = repo
	cmd.Env = env
	out, err := cmd.Output()
	if err == nil {
		t.Fatalf("%s %s succeeded, want failure\n%s", name, strings.Join(args, " "), out)
	}
	if err := json.Unmarshal(out, target); err != nil {
		t.Fatalf("%s %s returned invalid failure JSON: %v\n%s", name, strings.Join(args, " "), err, out)
	}
}

// Run runs a command and returns combined output.
func Run(t *testing.T, repo string, env []string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = repo
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, out)
	}
	return string(out)
}

// RunCommand runs a command and returns stdout, stderr, and error.
func RunCommand(repo string, env []string, name string, args ...string) ([]byte, []byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = repo
	cmd.Env = env
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.Output()
	return stdout, stderr.Bytes(), err
}

// ReadJSONFile decodes a JSON file into a typed value.
func ReadJSONFile[T any](t *testing.T, path string) T {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var value T
	if err := json.Unmarshal(content, &value); err != nil {
		t.Fatalf("decode %s: %v\n%s", path, err, content)
	}
	return value
}

// ReadTraceByStatus returns the single trace with the requested status.
func ReadTraceByStatus(t *testing.T, home string, status caltrace.Status) caltrace.Trace {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(home, "discovery"))
	if err != nil {
		t.Fatalf("read discovery directory: %v", err)
	}
	var matches []caltrace.Trace
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		trace := ReadJSONFile[caltrace.Trace](t, filepath.Join(home, "discovery", entry.Name(), "trace.json"))
		if trace.Status == status {
			matches = append(matches, trace)
		}
	}
	if len(matches) != 1 {
		t.Fatalf("traces with status %s = %#v, want one", status, matches)
	}
	return matches[0]
}

// FindProvider returns the provider with the requested name and kind.
func FindProvider(providers []ProviderSummary, name, kind string) (ProviderSummary, bool) {
	for _, provider := range providers {
		if provider.Name == name && provider.Kind == kind {
			return provider, true
		}
	}
	return ProviderSummary{}, false
}

// WriteConfig writes a minimal CAL config with one CLI directory and one app directory.
func WriteConfig(t *testing.T, path, cliDir, appDir string) {
	t.Helper()
	content := `{"provider_sources":[{"kind":"path","value":` + strconv.Quote(cliDir) + `},{"kind":"path","value":` + strconv.Quote(appDir) + `}]}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

// WriteFakeExecutable writes a fake PATH executable for smoke tests.
func WriteFakeExecutable(t *testing.T, path string) {
	t.Helper()
	script := `#!/bin/sh
if [ "$1" = "--help" ]; then
  echo "Fake Exporter"
  echo "CAL_CAPABILITY document.export_pdf"
  echo "CAL_COMMAND export-pdf --source {{source}} --target {{target}}"
  exit 0
fi
if [ "$1" = "export-pdf" ]; then
  source=""
  target=""
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --source)
        source="$2"
        shift 2
        ;;
      --target)
        target="$2"
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done
  if [ -z "$source" ] || [ -z "$target" ]; then
    exit 2
  fi
  ` + WriteParseablePDFCommand() + `
  exit $?
fi
exit 64
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake executable: %v", err)
	}
}

// WriteFakeExporter writes a CAL-marker fake exporter.
func WriteFakeExporter(t *testing.T, path, targetWriteCommand string) {
	t.Helper()
	script := `#!/bin/sh
if [ "$1" = "--help" ]; then
  echo "Fake Exporter"
  echo "CAL_CAPABILITY document.export_pdf"
  echo "CAL_COMMAND export-pdf --source {{source}} --target {{target}}"
  exit 0
fi
if [ "$1" = "export-pdf" ]; then
  source=""
  target=""
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --source)
        source="$2"
        shift 2
        ;;
      --target)
        target="$2"
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done
  if [ -z "$source" ] || [ -z "$target" ]; then
    exit 2
  fi
  ` + targetWriteCommand + `
  exit $?
fi
exit 64
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake exporter: %v", err)
	}
}

// WriteMarkerFreeFakeExporter writes a fake exporter without CAL markers.
func WriteMarkerFreeFakeExporter(t *testing.T, path, targetWriteCommand string) {
	t.Helper()
	script := `#!/bin/sh
if [ "$1" = "--help" ]; then
  echo "Fake Exporter 1.0"
  echo
  echo "Usage:"
  echo "  fake-exporter export-pdf --source <path> --target <path>"
  echo
  echo "Commands:"
  echo "  export-pdf    Convert a document to PDF"
  exit 0
fi
if [ "$1" = "export-pdf" ]; then
  source=""
  target=""
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --source)
        source="$2"
        shift 2
        ;;
      --target)
        target="$2"
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done
  if [ -z "$source" ] || [ -z "$target" ]; then
    exit 2
  fi
  ` + targetWriteCommand + `
  exit $?
fi
exit 64
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write marker-free fake exporter: %v", err)
	}
}

// WriteProposalBackedExporter writes a fake exporter for replay proposal tests.
func WriteProposalBackedExporter(t *testing.T, path string) {
	t.Helper()
	script := `#!/bin/sh
if [ "$1" = "--help" ]; then
  echo "Proposal Exporter 1.0"
  echo "Usage: proposal-exporter make-pdf --in <path> --out <path>"
  exit 0
fi
if [ "$1" = "make-pdf" ]; then
  source=""
  target=""
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --in)
        source="$2"
        shift 2
        ;;
      --out)
        target="$2"
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done
  if [ -z "$source" ] || [ -z "$target" ]; then
    exit 2
  fi
  ` + WriteParseablePDFCommand() + `
  exit $?
fi
exit 64
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write proposal-backed exporter: %v", err)
	}
}

// WriteReplayProposal writes a replay proposal fixture.
func WriteReplayProposal(t *testing.T, path string) string {
	t.Helper()
	content := `{
  "metadata": {"source": "replay", "prompt_version": "test-v1", "model": "fixture", "schema_version": "proposal.v1"},
  "verifier_packages": [{
    "id": "pdf_magic_check",
    "description": "Passes when the target artifact starts with a PDF marker.",
    "verify_py": ` + strconv.Quote(pdfMagicVerifierPY()) + `
  }],
  "candidates": [{
    "capability_id": "document.export_pdf",
    "description": "Export a document to a PDF artifact.",
    "execution": {
      "kind": "cli",
      "spec": {"args": ["make-pdf", "--in", "{{source}}", "--out", "{{target}}"]}
    },
    "rationale": "replayed proposal maps make-pdf to PDF export"
  }],
  "probe_plans": [{
    "candidate_index": 0,
    "inputs": {"target": "{{workdir}}/output.pdf"},
    "fixtures": [{"input": "source", "filename": "input.txt", "content": "hello\n"}],
    "verifier": {"id": "pdf_magic_check"},
    "rationale": "PDF parsing is deterministic evidence"
  }]
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write replay proposal: %v", err)
	}
	return path
}

// WriteReplayGeneratedVerifierProposal writes a replay proposal with a generated verifier.
func WriteReplayGeneratedVerifierProposal(t *testing.T, path string) string {
	t.Helper()
	verifyPY := pdfMagicVerifierPY()
	content := `{
  "metadata": {"source": "replay", "prompt_version": "test-v1", "model": "fixture", "schema_version": "proposal.v1"},
  "verifier_packages": [{
    "id": "pdf_magic_check",
    "description": "Passes when the target artifact starts with a PDF marker.",
    "verify_py": ` + strconv.Quote(verifyPY) + `
  }],
  "candidates": [{
    "capability_id": "document.export_pdf",
    "description": "Export a document to a PDF artifact.",
    "execution": {
      "kind": "cli",
      "spec": {"args": ["make-pdf", "--in", "{{source}}", "--out", "{{target}}"]}
    },
    "rationale": "replayed proposal maps make-pdf to PDF export"
  }],
  "probe_plans": [{
    "candidate_index": 0,
    "inputs": {"target": "{{workdir}}/output.pdf"},
    "fixtures": [{"input": "source", "filename": "input.txt", "content": "hello\n"}],
    "verifier": {"id": "pdf_magic_check"},
    "rationale": "the generated verifier checks the PDF marker"
  }]
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write generated verifier proposal: %v", err)
	}
	return path
}

func pdfMagicVerifierPY() string {
	return `import json
import os
import sys

def fail(code, message):
    print(json.dumps({"passed": False, "error": {"code": code, "message": message}}))
    sys.exit(0)

request = json.load(sys.stdin)
verifier_id = request["verifier"]["id"]
inputs = request.get("inputs") or {}
target = inputs.get("target")
if not isinstance(target, str) or not target:
    fail("missing_input", 'verifier input "target" is required')
if not os.path.exists(target):
    fail("file_missing", "target file is missing")
with open(target, "rb") as handle:
    content = handle.read(8)
if not content.startswith(b"%PDF"):
    fail("pdf_magic_missing", "target does not start with a PDF marker")
print(json.dumps({
    "passed": True,
    "evidence": [{
        "id": verifier_id,
        "type": verifier_id,
        "content": {"target": target},
    }],
    "outputs": {"target": target},
}))
`
}

// WriteParseablePDFCommand returns shell code that writes a minimal parseable PDF.
func WriteParseablePDFCommand() string {
	return `printf '%s\n' '%PDF-1.4' '1 0 obj' '<< /Type /Catalog /Pages 2 0 R >>' 'endobj' '2 0 obj' '<< /Type /Pages /Kids [3 0 R] /Count 1 >>' 'endobj' '3 0 obj' '<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] /Contents 4 0 R >>' 'endobj' '4 0 obj' '<< /Length 44 >>' 'stream' 'BT /F1 12 Tf 10 100 Td (fake pdf) Tj ET' 'endstream' 'endobj' 'xref' '0 5' '0000000000 65535 f ' '0000000009 00000 n ' '0000000058 00000 n ' '0000000115 00000 n ' '0000000202 00000 n ' 'trailer' '<< /Root 1 0 R /Size 5 >>' 'startxref' '295' '%%EOF' > "$target"`
}

// AssertPromotionAction checks the single trace promotion action.
func AssertPromotionAction(t *testing.T, home, traceID, capabilityAction, bindingAction string) {
	t.Helper()
	trace := ReadJSONFile[caltrace.Trace](t, filepath.Join(home, "discovery", traceID, "trace.json"))
	promotions := TracePromotions(trace)
	if len(promotions) != 1 {
		t.Fatalf("trace %s has no promotion", traceID)
	}
	if promotions[0].CapabilityAction != capabilityAction || promotions[0].BindingAction != bindingAction {
		t.Fatalf("trace promotion = %#v, want capability_action=%q binding_action=%q", promotions[0], capabilityAction, bindingAction)
	}
}

// TracePromotions returns trace promotions.
func TracePromotions(trace caltrace.Trace) []caltrace.Promotion {
	return trace.Promotions
}

// WriteMultiCapabilityExporter writes a fake multi-capability exporter.
func WriteMultiCapabilityExporter(t *testing.T, path string) {
	t.Helper()
	script := `#!/bin/sh
if [ "$1" = "--help" ]; then
  echo "Multi Exporter"
  echo "Usage: multi-exporter make-pdf --out <path>"
  echo "Usage: multi-exporter write-note --out <path>"
  exit 0
fi
if [ "$1" = "make-pdf" ]; then
  target=""
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --out)
        target="$2"
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done
  if [ -z "$target" ]; then
    exit 2
  fi
  ` + WriteParseablePDFCommand() + `
  exit $?
fi
if [ "$1" = "write-note" ]; then
  target=""
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --out)
        target="$2"
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done
  if [ -z "$target" ]; then
    exit 2
  fi
  printf 'hello\n' > "$target"
  exit $?
fi
exit 64
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write multi-capability exporter: %v", err)
	}
}

// WriteMultiCapabilityProposal writes a replay proposal with two capabilities.
func WriteMultiCapabilityProposal(t *testing.T, path string) string {
	t.Helper()
	content := `{
  "metadata": {"source": "replay", "prompt_version": "test-v1", "model": "fixture", "schema_version": "proposal.v1"},
  "verifier_packages": [
    {
      "id": "pdf_magic_check",
      "description": "Passes when the target artifact starts with a PDF marker.",
      "verify_py": ` + strconv.Quote(pdfMagicVerifierPY()) + `
    },
    {
      "id": "file_exists_check",
      "description": "Passes when the target file exists.",
      "verify_py": ` + strconv.Quote(fileExistsVerifierPY()) + `
    }
  ],
  "candidates": [
    {
      "capability_id": "document.export_pdf",
    "description": "Export a document to a PDF artifact.",
      "execution": {"kind": "cli", "spec": {"args": ["make-pdf", "--out", "{{target}}"]}},
      "rationale": "provider can create a PDF file"
    },
    {
      "capability_id": "text.write_file",
      "description": "Write a text file artifact.",
      "execution": {"kind": "cli", "spec": {"args": ["write-note", "--out", "{{target}}"]}},
      "rationale": "provider can write a text note"
    }
  ],
  "probe_plans": [
    {
      "candidate_index": 0,
      "inputs": {"target": "{{workdir}}/output.pdf"},
      "verifier": {"id": "pdf_magic_check"},
      "rationale": "PDF parser validates the generated file"
    },
    {
      "candidate_index": 1,
      "inputs": {"target": "{{workdir}}/note.txt"},
      "verifier": {"id": "file_exists_check"},
      "rationale": "file existence validates note creation"
    }
  ]
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write multi-capability proposal: %v", err)
	}
	return path
}

func fileExistsVerifierPY() string {
	return `import json
import os
import sys

request = json.load(sys.stdin)
verifier_id = request["verifier"]["id"]
target = (request.get("inputs") or {}).get("target")
if not isinstance(target, str) or not os.path.exists(target):
    print(json.dumps({"passed": False, "error": {"code": "file_missing", "message": "target file is missing"}}))
    sys.exit(0)
print(json.dumps({
    "passed": True,
    "evidence": [{
        "id": verifier_id,
        "type": verifier_id,
        "content": {"target": target},
    }],
    "outputs": {"target": target},
}))
`
}

// WriteLocalVerifier writes one verifier package under the CAL home for tests.
func WriteLocalVerifier(t *testing.T, home string, id string, verifyPY string) {
	t.Helper()
	dir := filepath.Join(home, "verifiers", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create local verifier dir: %v", err)
	}
	meta := map[string]any{
		"id":         id,
		"runtime":    "python3",
		"entry":      "verify.py",
		"timeout_ms": 3000,
	}
	content, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		t.Fatalf("encode local verifier metadata: %v", err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(filepath.Join(dir, "meta.json"), content, 0o644); err != nil {
		t.Fatalf("write local verifier metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "verify.py"), []byte(verifyPY), 0o644); err != nil {
		t.Fatalf("write local verifier script: %v", err)
	}
}

// WritePDFMagicVerifier writes a PDF marker verifier package under the CAL home.
func WritePDFMagicVerifier(t *testing.T, home string, id string) {
	t.Helper()
	WriteLocalVerifier(t, home, id, pdfMagicVerifierPY())
}

// WriteFileExistsVerifier writes a target-exists verifier package under the CAL home.
func WriteFileExistsVerifier(t *testing.T, home string, id string) {
	t.Helper()
	WriteLocalVerifier(t, home, id, fileExistsVerifierPY())
}

// WriteImageDimensionsVerifier writes an image dimensions verifier package under the CAL home.
func WriteImageDimensionsVerifier(t *testing.T, home string, id string) {
	t.Helper()
	WriteLocalVerifier(t, home, id, imageDimensionsVerifierPY())
}

func imageDimensionsVerifierPY() string {
	return `import json
import struct
import sys

request = json.load(sys.stdin)
verifier_id = request["verifier"]["id"]
inputs = request.get("inputs") or {}
target = inputs.get("target")
want_width = int(inputs.get("width"))
want_height = int(inputs.get("height"))

def fail(code, message):
    print(json.dumps({"passed": False, "error": {"code": code, "message": message}}))
    sys.exit(0)

with open(target, "rb") as handle:
    data = handle.read(24)
if len(data) < 24 or data[:8] != b"\x89PNG\r\n\x1a\n":
    fail("not_png", "target is not a PNG")
width, height = struct.unpack(">II", data[16:24])
if width != want_width or height != want_height:
    fail("dimension_mismatch", "target dimensions do not match requested dimensions")
print(json.dumps({
    "passed": True,
    "evidence": [{"id": verifier_id, "type": verifier_id, "content": {"target": target, "width": width, "height": height}}],
    "outputs": {"target": target, "width": width, "height": height},
}))
`
}

// Verifier returns a verifier by id.
func Verifier(id string) core.Verifier {
	return core.Verifier{ID: id}
}

// JSONArg marshals a value for an --inputs-json command argument.
func JSONArg(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("encode JSON arg: %v", err)
	}
	return string(encoded)
}

// PlistFixture returns a simple XML plist fixture.
func PlistFixture(name string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict><key>name</key><string>` + name + `</string></dict></plist>
`
}

// FileRunInputs creates a source fixture and target path input map.
func FileRunInputs(t *testing.T, root, name, sourceName, sourceContent, targetName string) map[string]any {
	t.Helper()
	dir := filepath.Join(root, name+"-run")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create run dir: %v", err)
	}
	source := filepath.Join(dir, sourceName)
	target := filepath.Join(dir, targetName)
	if err := os.WriteFile(source, []byte(sourceContent), 0o644); err != nil {
		t.Fatalf("write run source: %v", err)
	}
	return map[string]any{"source": source, "target": target}
}

// HasObservation reports whether a trace has an observation containing text.
func HasObservation(observations []caltrace.Observation, source, text string) bool {
	for _, observation := range observations {
		if observation.Source != source {
			continue
		}
		content, _ := observation.Content["text"].(string)
		if strings.Contains(content, text) {
			return true
		}
	}
	return false
}

// WritePNG writes a solid test PNG.
func WritePNG(t *testing.T, path string, width, height int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{G: 0xff, A: 0xff})
		}
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create PNG: %v", err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}
}

// AssertPNGDimensions checks a PNG file's dimensions.
func AssertPNGDimensions(t *testing.T, path string, width, height int) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open PNG: %v", err)
	}
	defer file.Close()
	config, err := png.DecodeConfig(file)
	if err != nil {
		t.Fatalf("decode PNG config: %v", err)
	}
	if config.Width != width || config.Height != height {
		t.Fatalf("PNG dimensions = %dx%d, want %dx%d", config.Width, config.Height, width, height)
	}
}

// WithHomeEnv returns env with the CAL home set to home.
func WithHomeEnv(env []string, home string) []string {
	return calpath.WithHomeEnv(env, home)
}

// HomeFromEnv returns the CAL home from an environment slice.
func HomeFromEnv(env []string) string {
	return calpath.HomeDirFromEnv(env)
}

// MapKeys returns sorted map keys.
func MapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// MissingInputs returns required names missing from inputs.
func MissingInputs(required []string, inputs map[string]any) []string {
	var missing []string
	for _, name := range required {
		value, ok := inputs[name]
		if !ok || value == nil || value == "" {
			missing = append(missing, name)
		}
	}
	return missing
}

// WriteJSONArtifact writes a pretty JSON artifact.
func WriteJSONArtifact(t *testing.T, path string, artifact any) {
	t.Helper()
	encoded, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		t.Fatalf("encode artifact: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create artifact dir: %v", err)
	}
	if err := os.WriteFile(path, append(encoded, '\n'), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
}
