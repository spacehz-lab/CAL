package rules

import (
	"fmt"
	"strings"

	"github.com/spacehz-lab/cal/internal/model"
)

const (
	SourceCLIHelpMarker          = "rules:cli_help_marker"
	SourceCLIHelpDocumentConvert = "rules:cli_help_document_convert"
	SourceCLIDocsCupsfilterPDF   = "rules:cli_docs_cupsfilter_pdf"
	SourceCLIHelpSipsResize      = "rules:cli_help_sips_resize"
)

const (
	capabilityDocumentConvert = "document.convert"
	capabilityImageResize     = "image.resize"

	inputSource = "source"
	inputTarget = "target"
	inputWidth  = "width"
	inputHeight = "height"

	paramFormat = "format"
)

func candidateFromHelp(providerID string, text string) (model.Candidate, bool, error) {
	capability, command, hasCapability, hasCommand := parseHelpMarkers(text)
	if hasCapability {
		if !hasCommand {
			return model.Candidate{}, false, nil
		}
		args := strings.Fields(command)
		if len(args) == 0 {
			return model.Candidate{}, false, fmt.Errorf("CAL_COMMAND marker is empty")
		}
		if !model.ValidCapabilityID(capability) {
			return model.Candidate{}, false, fmt.Errorf("capability id %q is invalid", capability)
		}
		return newCLICandidate(providerID, capability, descriptionForCapability(capability), args, SourceCLIHelpMarker), true, nil
	}

	if candidate, ok := sipsImageResizeFromHelp(providerID, text); ok {
		return candidate, true, nil
	}
	if candidate, ok := cupsfilterDocumentExportPDFFromHelp(providerID, text); ok {
		return candidate, true, nil
	}
	return markerFreeDocumentExportPDFFromHelp(providerID, text)
}

func sipsImageResizeFromHelp(providerID string, text string) (model.Candidate, bool) {
	normalized := strings.ToLower(text)
	if !strings.Contains(normalized, "sips") ||
		!strings.Contains(normalized, "--resampleheightwidth") ||
		!strings.Contains(normalized, "--out") {
		return model.Candidate{}, false
	}
	args := []string{"-z", "{{" + inputHeight + "}}", "{{" + inputWidth + "}}", "{{" + inputSource + "}}", "--out", "{{" + inputTarget + "}}"}
	return newCLICandidate(providerID, capabilityImageResize, descriptionForCapability(capabilityImageResize), args, SourceCLIHelpSipsResize), true
}

func cupsfilterDocumentExportPDFFromHelp(providerID string, text string) (model.Candidate, bool) {
	normalized := strings.ToLower(text)
	if !strings.Contains(normalized, "cupsfilter") ||
		!strings.Contains(normalized, "-i mime/type") ||
		!strings.Contains(normalized, "-m mime/type") ||
		!strings.Contains(normalized, "application/pdf") {
		return model.Candidate{}, false
	}
	candidate := newCLICandidate(
		providerID,
		capabilityDocumentConvert,
		descriptionForCapability(capabilityDocumentConvert),
		[]string{"-i", "text/plain", "-m", "application/pdf", "{{" + inputSource + "}}"},
		SourceCLIDocsCupsfilterPDF,
	)
	candidate.Execution.Spec[model.ExecutionSpecStdoutPathInput] = inputTarget
	return candidate, true
}

func markerFreeDocumentExportPDFFromHelp(providerID string, text string) (model.Candidate, bool, error) {
	normalized := strings.ToLower(text)
	if !strings.Contains(normalized, "export-pdf") ||
		!strings.Contains(normalized, "--source") ||
		!strings.Contains(normalized, "--target") {
		return model.Candidate{}, false, nil
	}
	args := []string{"export-pdf", "--source", "{{" + inputSource + "}}", "--target", "{{" + inputTarget + "}}"}
	return newCLICandidate(providerID, capabilityDocumentConvert, descriptionForCapability(capabilityDocumentConvert), args, SourceCLIHelpDocumentConvert), true, nil
}

func parseHelpMarkers(text string) (string, string, bool, bool) {
	var capability string
	var command string
	hasCapability := false
	hasCommand := false
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "CAL_CAPABILITY "):
			hasCapability = true
			capability = strings.TrimSpace(strings.TrimPrefix(line, "CAL_CAPABILITY "))
		case strings.HasPrefix(line, "CAL_COMMAND "):
			hasCommand = true
			command = strings.TrimSpace(strings.TrimPrefix(line, "CAL_COMMAND "))
		case line == "CAL_COMMAND":
			hasCommand = true
		}
	}
	return capability, command, hasCapability, hasCommand
}

func newCLICandidate(providerID string, capabilityID string, description string, args []string, source string) model.Candidate {
	return model.Candidate{
		ProviderID:   strings.TrimSpace(providerID),
		CapabilityID: capabilityID,
		Description:  description,
		Source:       source,
		Execution: model.Execution{
			Kind: model.ExecutionKindCLI,
			Spec: map[string]any{model.ExecutionSpecArgs: args},
		},
	}
}

func probeInputs(capabilityID string) map[string]any {
	switch capabilityID {
	case capabilityImageResize:
		return map[string]any{
			inputSource: "{{workdir}}/input.png",
			inputTarget: "{{workdir}}/output.png",
			inputWidth:  12,
			inputHeight: 8,
		}
	default:
		return map[string]any{
			inputSource: "{{workdir}}/input.txt",
			inputTarget: "{{workdir}}/output.pdf",
		}
	}
}

func verifyForCapability(capabilityID string) model.VerifySpec {
	format := "pdf"
	if capabilityID == capabilityImageResize {
		format = "png"
	}
	return model.VerifySpec{
		Level:  model.VerifyLevelL2,
		Method: model.VerifyMethodExecute,
		Checks: []model.VerifyCheck{{
			Subject:   model.VerifySubject{Type: model.VerifySubjectFile, Input: inputTarget},
			Predicate: model.VerifyPredicateFormat,
			Params:    map[string]any{paramFormat: format},
		}},
	}
}

func descriptionForCapability(capabilityID string) string {
	switch capabilityID {
	case capabilityDocumentConvert:
		return "Export or convert a document or text file into a PDF artifact."
	case capabilityImageResize:
		return "Resize an image artifact to requested dimensions."
	default:
		return "Execute the observed provider capability " + capabilityID + "."
	}
}
