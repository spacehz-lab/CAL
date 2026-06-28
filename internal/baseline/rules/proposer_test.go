package rules

import (
	"context"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/proposalflow"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

func TestProposerProposesCLIHelpMarkerCandidate(t *testing.T) {
	response, err := (Proposer{}).Propose(context.Background(), proposalflow.Request{
		Provider:    core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI},
		DebugFilter: "document.export_pdf",
		Observations: []caltrace.Observation{{
			Type: "cli_output",
			Content: map[string]any{
				"text": "Fake Exporter\nCAL_CAPABILITY document.export_pdf\nCAL_COMMAND export-pdf --source {{source}} --target {{target}}\n",
			},
		}},
	})
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}
	if len(response.Candidates) != 1 {
		t.Fatalf("candidates len = %d, want 1", len(response.Candidates))
	}
	candidate := response.Candidates[0]
	if candidate.ProviderID != "provider_cli" || candidate.CapabilityID != "document.export_pdf" {
		t.Fatalf("candidate = %#v, want provider and capability ids", candidate)
	}
	if candidate.Execution.Kind != core.ExecutionKindCLI {
		t.Fatalf("execution kind = %q, want cli", candidate.Execution.Kind)
	}
	args, ok := candidate.Execution.Spec["args"].([]string)
	if !ok {
		t.Fatalf("args = %#v, want []string", candidate.Execution.Spec["args"])
	}
	want := []string{"export-pdf", "--source", "{{source}}", "--target", "{{target}}"}
	if len(args) != len(want) {
		t.Fatalf("args len = %d, want %d", len(args), len(want))
	}
	for index := range want {
		if args[index] != want[index] {
			t.Fatalf("args[%d] = %q, want %q", index, args[index], want[index])
		}
	}
}

func TestProposerProposesCLIHelpMarkerCandidateWithoutHint(t *testing.T) {
	response, err := (Proposer{}).Propose(context.Background(), proposalflow.Request{
		Provider: core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI},
		Observations: []caltrace.Observation{{
			Type: "cli_output",
			Content: map[string]any{
				"text": "Fake Exporter\nCAL_CAPABILITY document.export_pdf\nCAL_COMMAND export-pdf --source {{source}} --target {{target}}\n",
			},
		}},
	})
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}
	if len(response.Candidates) != 1 || response.Candidates[0].CapabilityID != "document.export_pdf" {
		t.Fatalf("candidates = %#v, want document.export_pdf candidate", response.Candidates)
	}
}

func TestProposerProposesMarkerFreeDocumentExportPDFCandidate(t *testing.T) {
	response, err := (Proposer{}).Propose(context.Background(), proposalflow.Request{
		Provider:    core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI},
		DebugFilter: "document.export_pdf",
		Observations: []caltrace.Observation{{
			Type: "cli_output",
			Content: map[string]any{
				"text": "Fake Exporter 1.0\n\nUsage:\n  fake-exporter export-pdf --source <path> --target <path>\n\nCommands:\n  export-pdf    Convert a document to PDF\n",
			},
		}},
	})
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}
	if len(response.Candidates) != 1 {
		t.Fatalf("candidates len = %d, want 1", len(response.Candidates))
	}
	candidate := response.Candidates[0]
	if candidate.ProviderID != "provider_cli" || candidate.CapabilityID != "document.export_pdf" {
		t.Fatalf("candidate = %#v, want provider and capability ids", candidate)
	}
	if candidate.Source != "rules:cli_help_export_pdf" {
		t.Fatalf("candidate source = %q, want marker-free export-pdf source", candidate.Source)
	}
	args, ok := candidate.Execution.Spec["args"].([]string)
	if !ok {
		t.Fatalf("args = %#v, want []string", candidate.Execution.Spec["args"])
	}
	want := []string{"export-pdf", "--source", "{{source}}", "--target", "{{target}}"}
	if len(args) != len(want) {
		t.Fatalf("args len = %d, want %d", len(args), len(want))
	}
	for index := range want {
		if args[index] != want[index] {
			t.Fatalf("args[%d] = %q, want %q", index, args[index], want[index])
		}
	}
}

func TestProposerProposesMarkerFreeDocumentExportPDFCandidateWithoutHint(t *testing.T) {
	response, err := (Proposer{}).Propose(context.Background(), proposalflow.Request{
		Provider: core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI},
		Observations: []caltrace.Observation{{
			Type: "cli_output",
			Content: map[string]any{
				"text": "Usage:\n  fake-exporter export-pdf --source <path> --target <path>\n",
			},
		}},
	})
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}
	if len(response.Candidates) != 1 || response.Candidates[0].CapabilityID != "document.export_pdf" {
		t.Fatalf("candidates = %#v, want document.export_pdf candidate", response.Candidates)
	}
}

func TestProposerProposesCupsfilterDocumentExportPDFCandidate(t *testing.T) {
	response, err := (Proposer{}).Propose(context.Background(), proposalflow.Request{
		Provider:    core.Provider{ID: "provider_cupsfilter", Kind: core.ProviderKindCLI},
		DebugFilter: "document.export_pdf",
		Observations: []caltrace.Observation{{
			Type:   "cli_output",
			Source: "man",
			Content: map[string]any{
				"text": "cupsfilter - convert a file to another format using cups filters\n\nSYNOPSIS\n  cupsfilter [ -i mime/type ] [ -m mime/type ] filename\n\nDESCRIPTION\n  By default, cupsfilter generates a PDF file. The converted file is sent to the standard output.\n\nOPTIONS\n  -m mime/type Specifies the destination file type. The default file type is application/pdf.\n",
			},
		}},
	})
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}
	if len(response.Candidates) != 1 {
		t.Fatalf("candidates len = %d, want 1", len(response.Candidates))
	}
	candidate := response.Candidates[0]
	if candidate.ProviderID != "provider_cupsfilter" || candidate.CapabilityID != "document.export_pdf" {
		t.Fatalf("candidate = %#v, want provider and capability ids", candidate)
	}
	if candidate.Source != "rules:cli_docs_cupsfilter_pdf" {
		t.Fatalf("candidate source = %q, want cupsfilter docs source", candidate.Source)
	}
	args, ok := candidate.Execution.Spec[core.ExecutionSpecArgs].([]string)
	if !ok {
		t.Fatalf("args = %#v, want []string", candidate.Execution.Spec[core.ExecutionSpecArgs])
	}
	want := []string{"-i", "text/plain", "-m", "application/pdf", "{{source}}"}
	if len(args) != len(want) {
		t.Fatalf("args len = %d, want %d", len(args), len(want))
	}
	for index := range want {
		if args[index] != want[index] {
			t.Fatalf("args[%d] = %q, want %q", index, args[index], want[index])
		}
	}
	if candidate.Execution.Spec[core.ExecutionSpecStdoutPathInput] != "target" {
		t.Fatalf("stdout path input = %#v, want target", candidate.Execution.Spec[core.ExecutionSpecStdoutPathInput])
	}
}

func TestProposerProposesCupsfilterDocumentExportPDFCandidateWithoutHint(t *testing.T) {
	response, err := (Proposer{}).Propose(context.Background(), proposalflow.Request{
		Provider: core.Provider{ID: "provider_cupsfilter", Kind: core.ProviderKindCLI},
		Observations: []caltrace.Observation{{
			Type:   "cli_output",
			Source: "man",
			Content: map[string]any{
				"text": "cupsfilter - convert a file to another format using cups filters\n\nSYNOPSIS\n  cupsfilter [ -i mime/type ] [ -m mime/type ] filename\n\nOPTIONS\n  -m mime/type Specifies the destination file type. The default file type is application/pdf.\n",
			},
		}},
	})
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}
	if len(response.Candidates) != 1 || response.Candidates[0].CapabilityID != "document.export_pdf" {
		t.Fatalf("candidates = %#v, want document.export_pdf candidate", response.Candidates)
	}
}

func TestProposerProposesSipsImageResizeCandidate(t *testing.T) {
	response, err := (Proposer{}).Propose(context.Background(), proposalflow.Request{
		Provider:    core.Provider{ID: "provider_sips", Kind: core.ProviderKindCLI},
		DebugFilter: "image.resize",
		Observations: []caltrace.Observation{{
			Type:   "cli_output",
			Source: "help",
			Content: map[string]any{
				"text": "sips - scriptable image processing system.\n\nImage modification functions:\n    -z, --resampleHeightWidth pixelsH pixelsW\n    -o, --out file-or-directory\n",
			},
		}},
	})
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}
	if len(response.Candidates) != 1 {
		t.Fatalf("candidates len = %d, want 1", len(response.Candidates))
	}
	candidate := response.Candidates[0]
	if candidate.ProviderID != "provider_sips" || candidate.CapabilityID != "image.resize" {
		t.Fatalf("candidate = %#v, want provider and capability ids", candidate)
	}
	if candidate.Source != "rules:cli_help_sips_resize" {
		t.Fatalf("candidate source = %q, want sips source", candidate.Source)
	}
	args, ok := candidate.Execution.Spec[core.ExecutionSpecArgs].([]string)
	if !ok {
		t.Fatalf("args = %#v, want []string", candidate.Execution.Spec[core.ExecutionSpecArgs])
	}
	want := []string{"-z", "{{height}}", "{{width}}", "{{source}}", "--out", "{{target}}"}
	if len(args) != len(want) {
		t.Fatalf("args len = %d, want %d", len(args), len(want))
	}
	for index := range want {
		if args[index] != want[index] {
			t.Fatalf("args[%d] = %q, want %q", index, args[index], want[index])
		}
	}
}

func TestProposerProposesSipsImageResizeCandidateWithoutHint(t *testing.T) {
	response, err := (Proposer{}).Propose(context.Background(), proposalflow.Request{
		Provider: core.Provider{ID: "provider_sips", Kind: core.ProviderKindCLI},
		Observations: []caltrace.Observation{{
			Type:   "cli_output",
			Source: "help",
			Content: map[string]any{
				"text": "sips\n-z, --resampleHeightWidth pixelsH pixelsW\n-o, --out file-or-directory\n",
			},
		}},
	})
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}
	if len(response.Candidates) != 1 || response.Candidates[0].CapabilityID != "image.resize" {
		t.Fatalf("candidates = %#v, want image.resize candidate", response.Candidates)
	}
}

func TestProposerIgnoresCapabilityMismatch(t *testing.T) {
	response, err := (Proposer{}).Propose(context.Background(), proposalflow.Request{
		Provider:    core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI},
		DebugFilter: "document.export_pdf",
		Observations: []caltrace.Observation{{
			Type: "cli_output",
			Content: map[string]any{
				"text": "CAL_CAPABILITY media.convert\nCAL_COMMAND convert --source {{source}} --target {{target}}\n",
			},
		}},
	})
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}
	if len(response.Candidates) != 0 {
		t.Fatalf("candidates len = %d, want 0", len(response.Candidates))
	}
}

func TestProposerIgnoresSipsHelpForDifferentCapability(t *testing.T) {
	response, err := (Proposer{}).Propose(context.Background(), proposalflow.Request{
		Provider:    core.Provider{ID: "provider_sips", Kind: core.ProviderKindCLI},
		DebugFilter: "document.export_pdf",
		Observations: []caltrace.Observation{{
			Type: "cli_output",
			Content: map[string]any{
				"text": "sips\n-z, --resampleHeightWidth pixelsH pixelsW\n-o, --out file-or-directory\n",
			},
		}},
	})
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}
	if len(response.Candidates) != 0 {
		t.Fatalf("candidates len = %d, want 0", len(response.Candidates))
	}
}

func TestProposerIgnoresMarkerFreeHelpForDifferentCapability(t *testing.T) {
	response, err := (Proposer{}).Propose(context.Background(), proposalflow.Request{
		Provider:    core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI},
		DebugFilter: "media.convert",
		Observations: []caltrace.Observation{{
			Type: "cli_output",
			Content: map[string]any{
				"text": "Usage:\n  fake-exporter export-pdf --source <path> --target <path>\n",
			},
		}},
	})
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}
	if len(response.Candidates) != 0 {
		t.Fatalf("candidates len = %d, want 0", len(response.Candidates))
	}
}

func TestProposerIgnoresIncompleteMarkerFreeExportPDFHelp(t *testing.T) {
	response, err := (Proposer{}).Propose(context.Background(), proposalflow.Request{
		Provider:    core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI},
		DebugFilter: "document.export_pdf",
		Observations: []caltrace.Observation{{
			Type: "cli_output",
			Content: map[string]any{
				"text": "Usage:\n  fake-exporter export-pdf --source <path>\n",
			},
		}},
	})
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}
	if len(response.Candidates) != 0 {
		t.Fatalf("candidates len = %d, want 0", len(response.Candidates))
	}
}

func TestProposerRejectsEmptyCommandMarker(t *testing.T) {
	_, err := (Proposer{}).Propose(context.Background(), proposalflow.Request{
		Provider:    core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI},
		DebugFilter: "document.export_pdf",
		Observations: []caltrace.Observation{{
			Type: "cli_output",
			Content: map[string]any{
				"text": "CAL_CAPABILITY document.export_pdf\nCAL_COMMAND \n",
			},
		}},
	})
	if err == nil {
		t.Fatal("Propose() error = nil, want empty command rejection")
	}
}
