package e2e

import (
	"os"
	"testing"
)

func writeReplayProposal(t *testing.T, path string) string {
	t.Helper()
	content := `{
  "metadata": {"source": "replay", "prompt_version": "test-v1", "model": "fixture", "schema_version": "proposal.v1"},
  "candidates": [{
    "capability_id": "document.convert",
    "description": "Export a document to a PDF artifact.",
    "execution": {
      "kind": "cli",
      "spec": {"args": ["make-pdf", "--in", "{{source}}", "--out", "{{target}}"]}
    }
  }],
  "probe_plans": [{
    "candidate_index": 0,
    "inputs": {"target": "{{workdir}}/output.pdf"},
    "fixtures": [{"input": "source", "filename": "input.txt", "content": "hello\n"}],
    "verify": {"level":"L2","method":"execute","checks":[{"subject":{"type":"file","input":"target"},"predicate":"format","params":{"format":"pdf"}}]}
  }]
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write replay proposal: %v", err)
	}
	return path
}

func writeLiveLLMSeedConvertProposal(t *testing.T, path string, capabilityID string) string {
	t.Helper()
	content := `{
  "metadata": {"source": "replay", "prompt_version": "test-v1", "model": "fixture", "schema_version": "proposal.v1"},
  "candidates": [{
    "capability_id": "` + capabilityID + `",
    "description": "Convert text into another document representation.",
    "execution": {
      "kind": "cli",
      "spec": {"args": ["make-pdf", "--in", "{{source}}", "--out", "{{target}}"]}
    }
  }],
  "probe_plans": [{
    "candidate_index": 0,
    "inputs": {"target": "{{workdir}}/output.pdf"},
    "fixtures": [{"input": "source", "filename": "input.txt", "content": "hello\n"}],
    "verify": {"level":"L2","method":"execute","checks":[{"subject":{"type":"file","input":"target"},"predicate":"format","params":{"format":"pdf"}}]}
  }]
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write live llm seed proposal: %v", err)
	}
	return path
}

func writeLiveLLMExporter(t *testing.T, path string) {
	t.Helper()
	script := `#!/bin/sh
if [ "$1" = "--help" ] || [ "$1" = "help" ]; then
  echo "Live LLM Exporter"
  echo ""
  echo "Usage:"
  echo "  live-llm-exporter <command> [options]"
  echo ""
  echo "Commands:"
  echo "  make-pdf    Convert a UTF-8 text file into a valid PDF document."
  echo ""
  echo "Command usage:"
  echo "  live-llm-exporter make-pdf --in <input.txt> --out <output.pdf>"
  echo ""
  echo "Options for make-pdf:"
  echo "  --in <input.txt>      Path to a UTF-8 text input file."
  echo "  --out <output.pdf>    Path where the generated PDF should be written."
  echo "  -h, --help            Show command help."
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
  ` + writeParseablePDFCommand() + `
  exit $?
fi
exit 64
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write live llm exporter: %v", err)
	}
}

func writeLiveLLMMultiCapabilityExporter(t *testing.T, path string) {
	t.Helper()
	script := `#!/bin/sh
if [ "$1" = "--help" ] || [ "$1" = "help" ]; then
  echo "Live Multi Tool"
  echo ""
  echo "Usage:"
  echo "  live-multi-tool <command> [options]"
  echo ""
  echo "Commands:"
  echo "  make-pdf      Convert a UTF-8 text file into a valid PDF document."
  echo "  write-note    Write a UTF-8 text note file."
  echo ""
  echo "Command usage:"
  echo "  live-multi-tool make-pdf --in <input.txt> --out <output.pdf>"
  echo "  live-multi-tool write-note --in <input.txt> --out <output.txt>"
  echo ""
  echo "Options:"
  echo "  --in <input.txt>       Path to a UTF-8 input file."
  echo "  --out <output-path>    Path where the command should write its output."
  echo "  -h, --help             Show command help."
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
  if [ -z "$target" ]; then
    exit 2
  fi
  ` + writeParseablePDFCommand() + `
  exit $?
fi
if [ "$1" = "write-note" ]; then
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
  if [ -z "$target" ]; then
    exit 2
  fi
  if [ -n "$source" ] && [ -f "$source" ]; then
    cat "$source" > "$target"
  else
    printf 'hello\n' > "$target"
  fi
  exit $?
fi
exit 64
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write live llm multi-capability exporter: %v", err)
	}
}

func writeLiveLLMMarkerWriter(t *testing.T, path string) {
	t.Helper()
	script := `#!/bin/sh
if [ "$1" = "--help" ] || [ "$1" = "help" ]; then
  echo "Live Marker Tool"
  echo ""
  echo "Usage:"
  echo "  live-marker-tool <command> [options]"
  echo ""
  echo "Commands:"
  echo "  write-marker    Writes the literal text CAL_PROBE_OK to the output file."
  echo ""
  echo "Command usage:"
  echo "  live-marker-tool write-marker --in <input.txt> --out <output.txt>"
  echo ""
  echo "Options for write-marker:"
  echo "  --in <input.txt>       Optional input file path."
  echo "  --out <output.txt>     Path where the marker output should be written."
  echo "  -h, --help             Show command help."
  echo ""
  echo "Output contract:"
  echo "  The output file contains CAL_PROBE_OK."
  exit 0
fi
if [ "$1" = "write-marker" ]; then
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
  printf 'CAL_PROBE_OK\n' > "$target"
  exit $?
fi
exit 64
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write live llm marker writer: %v", err)
	}
}

func writeLiveLLMNestedPackageManager(t *testing.T, path string) {
	t.Helper()
	script := `#!/bin/sh
if [ "$1" = "--help" ] || [ "$1" = "help" ]; then
  echo "Live Nested Package Manager"
  echo ""
  echo "Usage:"
  echo "  live-nested-pm <namespace> <command> [options]"
  echo ""
  echo "Command usage:"
  echo "  live-nested-pm package install <name>"
  echo "  live-nested-pm package remove <name>"
  echo "  live-nested-pm package update"
  echo "  live-nested-pm package upgrade"
  echo "  live-nested-pm system doctor --json --output <report.json>"
  echo ""
  echo "Commands:"
  echo "  package install <name>   Install a package. Modifies local package state and may contact remote registries."
  echo "  package remove <name>    Remove an installed package. Modifies local package state."
  echo "  package update           Refresh package metadata from remote registries and modifies local metadata state."
  echo "  package upgrade          Upgrade installed packages and modifies local package state."
  echo "  system doctor --json --output <report.json>  Validate local package-manager configuration without changing state. Writes a JSON report with status=\"ok\" and checks[]."
  echo ""
  echo "Options for system doctor:"
  echo "  --json                    Write the report as JSON."
  echo "  --output <report.json>    Path where the JSON report should be written."
  echo "  -h, --help                Show command help."
  exit 0
fi
if [ "$1" = "system" ] && [ "$2" = "doctor" ]; then
  target=""
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --output)
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
  printf '%s\n' '{"status":"ok","checks":[{"name":"prefix","result":"ok"},{"name":"cache","result":"ok"}]}' > "$target"
  printf '%s\n' "wrote $target"
  exit $?
fi
if [ "$1" = "package" ]; then
  case "$2" in
    install|remove|update|upgrade)
      printf '%s\n' "refusing to run state-changing package command in test fixture" >&2
      exit 70
      ;;
  esac
fi
exit 64
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write live llm nested package manager: %v", err)
	}
}

func writeParseablePDFCommand() string {
	return `printf '%s\n' '%PDF-1.4' '1 0 obj' '<< /Type /Catalog /Pages 2 0 R >>' 'endobj' '2 0 obj' '<< /Type /Pages /Kids [3 0 R] /Count 1 >>' 'endobj' '3 0 obj' '<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] /Contents 4 0 R >>' 'endobj' '4 0 obj' '<< /Length 44 >>' 'stream' 'BT /F1 12 Tf 10 100 Td (fake pdf) Tj ET' 'endstream' 'endobj' 'xref' '0 5' '0000000000 65535 f ' '0000000009 00000 n ' '0000000058 00000 n ' '0000000115 00000 n ' '0000000202 00000 n ' 'trailer' '<< /Root 1 0 R /Size 5 >>' 'startxref' '295' '%%EOF' > "$target"`
}
