package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func writeAcquisitionScript(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake-cli")
	script := `#!/bin/sh
if [ "$1" = "--help" ]; then
  echo "CAL_CAPABILITY document.export_pdf"
  echo "CAL_COMMAND export-pdf --source {{source}} --target {{target}}"
  exit 0
fi
if [ "$1" = "export-pdf" ]; then
  target=""
  while [ "$#" -gt 0 ]; do
    if [ "$1" = "--target" ]; then
      target="$2"
      break
    fi
    shift
  done
  if [ -z "$target" ]; then
    exit 2
  fi
  printf '%s\n' '%PDF-1.4' '1 0 obj' '<< /Type /Catalog /Pages 2 0 R >>' 'endobj' '2 0 obj' '<< /Type /Pages /Kids [3 0 R] /Count 1 >>' 'endobj' '3 0 obj' '<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] /Contents 4 0 R >>' 'endobj' '4 0 obj' '<< /Length 44 >>' 'stream' 'BT /F1 12 Tf 10 100 Td (fake pdf) Tj ET' 'endstream' 'endobj' 'xref' '0 5' '0000000000 65535 f ' '0000000009 00000 n ' '0000000058 00000 n ' '0000000115 00000 n ' 'trailer' '<< /Root 1 0 R /Size 5 >>' 'startxref' '295' '%%EOF' > "$target"
  exit 0
fi
exit 64
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write acquisition script: %v", err)
	}
	return path
}

func writeProposalBackedScript(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "proposal-cli")
	script := `#!/bin/sh
if [ "$1" = "--help" ]; then
  echo "Proposal CLI"
  echo "Usage: proposal-cli make-pdf --in <path> --out <path>"
  exit 0
fi
if [ "$1" = "make-pdf" ]; then
  target=""
  while [ "$#" -gt 0 ]; do
    if [ "$1" = "--out" ]; then
      target="$2"
      break
    fi
    shift
  done
  if [ -z "$target" ]; then
    exit 2
  fi
  printf '%s\n' '%PDF-1.4' '1 0 obj' '<< /Type /Catalog /Pages 2 0 R >>' 'endobj' '2 0 obj' '<< /Type /Pages /Kids [3 0 R] /Count 1 >>' 'endobj' '3 0 obj' '<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] /Contents 4 0 R >>' 'endobj' '4 0 obj' '<< /Length 44 >>' 'stream' 'BT /F1 12 Tf 10 100 Td (fake pdf) Tj ET' 'endstream' 'endobj' 'xref' '0 5' '0000000000 65535 f ' '0000000009 00000 n ' '0000000058 00000 n ' '0000000115 00000 n ' 'trailer' '<< /Root 1 0 R /Size 5 >>' 'startxref' '295' '%%EOF' > "$target"
  exit 0
fi
exit 64
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write proposal-backed script: %v", err)
	}
	return path
}

func writeDiscoveryProposal(t *testing.T, providerID string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "proposal.json")
	providerField := ""
	if providerID != "" {
		providerField = `"provider_id": "` + providerID + `",`
	}
	content := `{
  "metadata": {"source": "replay"},
  "candidates": [{
    ` + providerField + `
    "capability_id": "document.export_pdf",
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
    "verify": {"level":"L2","method":"execute","checks":[{"subject":"target","predicate":"format","params":{"format":"pdf"}}]}
  }]
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write proposal: %v", err)
	}
	return path
}
