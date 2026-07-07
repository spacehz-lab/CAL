package rules

import (
	"context"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/internal/proposal"
)

func TestRunnerMatchesMarkerRule(t *testing.T) {
	result, err := NewRunner().Run(context.Background(), request("Fake Exporter\nCAL_CAPABILITY document.convert\nCAL_COMMAND export-pdf --source {{source}} --target {{target}}\n", ""))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(result.Candidates) != 1 || result.Candidates[0].Source != SourceCLIHelpMarker {
		t.Fatalf("candidates = %#v, want marker candidate", result.Candidates)
	}
	args := result.Candidates[0].Execution.Spec[model.ExecutionSpecArgs].([]string)
	if len(args) != 5 || args[0] != "export-pdf" {
		t.Fatalf("args = %#v, want marker command args", args)
	}
	if result.ProbePlans[0].Verify.Level != model.VerifyLevelL2 {
		t.Fatalf("verify = %#v, want L2", result.ProbePlans[0].Verify)
	}
}

func TestRunnerMatchesCupsfilterRule(t *testing.T) {
	help := `cupsfilter - convert a file to another format using cups filters
SYNOPSIS
  cupsfilter [ -i mime/type ] [ -m mime/type ] filename
OPTIONS
  -m mime/type Specifies the destination file type. The default file type is application/pdf.
`
	result, err := NewRunner().Run(context.Background(), request(help, "convert a document"))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	candidate := result.Candidates[0]
	if candidate.CapabilityID != capabilityDocumentConvert || candidate.Source != SourceCLIDocsCupsfilterPDF {
		t.Fatalf("candidate = %#v, want cupsfilter document.convert", candidate)
	}
	if candidate.Execution.Spec[model.ExecutionSpecStdoutPathInput] != inputTarget {
		t.Fatalf("execution = %#v, want stdout target", candidate.Execution)
	}
}

func TestRunnerMatchesSipsRule(t *testing.T) {
	help := "sips - scriptable image processing system\n-z, --resampleHeightWidth pixelsH pixelsW\n-o, --out file-or-directory\n"
	result, err := NewRunner().Run(context.Background(), request(help, "resize an image"))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	candidate := result.Candidates[0]
	if candidate.CapabilityID != capabilityImageResize || candidate.Source != SourceCLIHelpSipsResize {
		t.Fatalf("candidate = %#v, want sips image.resize", candidate)
	}
	if len(result.ProbePlans[0].Fixtures) != 1 || result.ProbePlans[0].Verify.Checks[0].Params[paramFormat] != "png" {
		t.Fatalf("probe plan = %#v, want png fixture and verify", result.ProbePlans[0])
	}
}

func TestRunnerRejectsNoMatch(t *testing.T) {
	_, err := NewRunner().Run(context.Background(), request("unknown help", ""))
	if err == nil {
		t.Fatal("Run() error = nil, want no match error")
	}
}

func TestRunnerRejectsEmptyMarkerCommand(t *testing.T) {
	_, err := NewRunner().Run(context.Background(), request("CAL_CAPABILITY document.convert\nCAL_COMMAND\n", ""))
	if err == nil {
		t.Fatal("Run() error = nil, want empty marker command error")
	}
}

func request(text string, hint string) *proposal.Request {
	return &proposal.Request{
		Provider: &model.Provider{ID: "provider_cli", Kind: model.ProviderKindCLI, Path: "/bin/tool"},
		Observations: []model.Observation{{
			ProviderID: "provider_cli",
			Type:       observationTypeCLIOutput,
			Source:     "help",
			Content:    map[string]any{observationContentText: text},
		}},
		Hint: hint,
	}
}
