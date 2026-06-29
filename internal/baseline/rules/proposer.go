package rules

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/proposal"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

// Proposer is the deterministic rules-based candidate proposer.
type Proposer struct{}

// Propose returns rule-derived candidates for the current skeleton.
func (Proposer) Propose(_ context.Context, request proposal.Request) (proposal.Result, error) {
	var candidates []caltrace.Candidate
	for _, observation := range request.Observations {
		if observation.Type != "cli_output" {
			continue
		}
		text, ok := observation.Content["text"].(string)
		if !ok {
			continue
		}
		candidate, ok, err := candidateFromHelp(request.Provider.ID, request.DebugFilter, text)
		if err != nil {
			return proposal.Result{}, err
		}
		if ok {
			candidates = append(candidates, candidate)
		}
	}
	return proposal.Select(rulesResult(candidates), proposal.SelectOptions{
		ProviderID:  request.Provider.ID,
		DebugFilter: request.DebugFilter,
	})
}

func rulesResult(candidates []caltrace.Candidate) proposal.Result {
	probePlans := make([]proposal.ProbePlan, 0, len(candidates))
	for index, candidate := range candidates {
		probePlans = append(probePlans, proposal.ProbePlan{
			CandidateIndex: index,
			Inputs:         probeInputs(candidate.CapabilityID),
			Fixtures:       probeFixtures(candidate.CapabilityID),
			Verify:         verifyForCapability(candidate.CapabilityID),
		})
	}
	return proposal.Result{
		Candidates: candidates,
		ProbePlans: probePlans,
	}
}

func probeInputs(capabilityID string) map[string]any {
	switch capabilityID {
	case "image.resize":
		return map[string]any{
			"source": "{{workdir}}/input.png",
			"target": "{{workdir}}/output.png",
			"width":  12,
			"height": 8,
		}
	default:
		return map[string]any{
			"source": "{{workdir}}/input.txt",
			"target": "{{workdir}}/output.pdf",
		}
	}
}

func probeFixtures(capabilityID string) []proposal.Fixture {
	switch capabilityID {
	case "document.export_pdf":
		return []proposal.Fixture{{
			Input:    "source",
			Filename: "input.txt",
			Content:  "cal probe input\n",
		}}
	case "image.resize":
		return []proposal.Fixture{{
			Input:    "source",
			Filename: "input.png",
			Content:  mustDecodeString(testPNGBase64),
		}}
	default:
		return nil
	}
}

func verifyForCapability(capabilityID string) core.VerifySpec {
	switch capabilityID {
	case "image.resize":
		return core.VerifySpec{Level: core.VerifyLevelL2, Method: core.VerifyMethodExecute, Checks: []core.VerifyCheck{{Subject: "target", Predicate: core.VerifyPredicateFormat, Params: map[string]any{"format": "png"}}}}
	default:
		return core.VerifySpec{Level: core.VerifyLevelL2, Method: core.VerifyMethodExecute, Checks: []core.VerifyCheck{{Subject: "target", Predicate: core.VerifyPredicateFormat, Params: map[string]any{"format": "pdf"}}}}
	}
}

func candidateFromHelp(providerID, capabilityID, text string) (caltrace.Candidate, bool, error) {
	capability, command, hasCapability, hasCommand := parseHelpMarkers(text)
	if hasCapability && !matchesCapabilityHint(capabilityID, capability) {
		return caltrace.Candidate{}, false, nil
	}
	if hasCapability {
		if !hasCommand {
			return caltrace.Candidate{}, false, nil
		}
		args := strings.Fields(command)
		if len(args) == 0 {
			return caltrace.Candidate{}, false, fmt.Errorf("CAL_COMMAND marker is empty")
		}
		return newCLIHelpCandidate(providerID, capability, descriptionForCapability(capability), args, "rules:cli_help_marker"), true, nil
	}

	if candidate, ok, err := sipsImageResizeFromHelp(providerID, capabilityID, text); ok || err != nil {
		return candidate, ok, err
	}
	if candidate, ok, err := cupsfilterDocumentExportPDFFromHelp(providerID, capabilityID, text); ok || err != nil {
		return candidate, ok, err
	}
	return markerFreeDocumentExportPDFFromHelp(providerID, capabilityID, text)
}

func sipsImageResizeFromHelp(providerID, capabilityID, text string) (caltrace.Candidate, bool, error) {
	const proposedCapability = "image.resize"
	if !matchesCapabilityHint(capabilityID, proposedCapability) {
		return caltrace.Candidate{}, false, nil
	}
	normalized := strings.ToLower(text)
	if !strings.Contains(normalized, "sips") ||
		!strings.Contains(normalized, "--resampleheightwidth") ||
		!strings.Contains(normalized, "--out") {
		return caltrace.Candidate{}, false, nil
	}
	args := []string{"-z", "{{height}}", "{{width}}", "{{source}}", "--out", "{{target}}"}
	return newCLIHelpCandidate(providerID, proposedCapability, descriptionForCapability(proposedCapability), args, "rules:cli_help_sips_resize"), true, nil
}

func cupsfilterDocumentExportPDFFromHelp(providerID, capabilityID, text string) (caltrace.Candidate, bool, error) {
	const proposedCapability = "document.export_pdf"
	if !matchesCapabilityHint(capabilityID, proposedCapability) {
		return caltrace.Candidate{}, false, nil
	}
	normalized := strings.ToLower(text)
	if !strings.Contains(normalized, "cupsfilter") ||
		!strings.Contains(normalized, "-i mime/type") ||
		!strings.Contains(normalized, "-m mime/type") ||
		!strings.Contains(normalized, "application/pdf") {
		return caltrace.Candidate{}, false, nil
	}
	args := []string{"-i", "text/plain", "-m", "application/pdf", "{{source}}"}
	candidate := newCLIHelpCandidate(providerID, proposedCapability, descriptionForCapability(proposedCapability), args, "rules:cli_docs_cupsfilter_pdf")
	candidate.Execution.Spec[core.ExecutionSpecStdoutPathInput] = "target"
	return candidate, true, nil
}

func markerFreeDocumentExportPDFFromHelp(providerID, capabilityID, text string) (caltrace.Candidate, bool, error) {
	const proposedCapability = "document.export_pdf"
	if !matchesCapabilityHint(capabilityID, proposedCapability) {
		return caltrace.Candidate{}, false, nil
	}
	normalized := strings.ToLower(text)
	if !strings.Contains(normalized, "export-pdf") ||
		!strings.Contains(normalized, "--source") ||
		!strings.Contains(normalized, "--target") {
		return caltrace.Candidate{}, false, nil
	}
	args := []string{"export-pdf", "--source", "{{source}}", "--target", "{{target}}"}
	return newCLIHelpCandidate(providerID, proposedCapability, descriptionForCapability(proposedCapability), args, "rules:cli_help_export_pdf"), true, nil
}

func matchesCapabilityHint(hint, capabilityID string) bool {
	return hint == "" || hint == capabilityID
}

func newCLIHelpCandidate(providerID, capabilityID, description string, args []string, source string) caltrace.Candidate {
	return caltrace.Candidate{
		ProviderID:   providerID,
		CapabilityID: capabilityID,
		Description:  description,
		Source:       source,
		Execution: core.Execution{
			Kind: core.ExecutionKindCLI,
			Spec: map[string]any{
				core.ExecutionSpecArgs: args,
			},
		},
	}
}

func descriptionForCapability(capabilityID string) string {
	switch capabilityID {
	case "document.export_pdf":
		return "Export or convert a document or text file into a PDF artifact."
	case "image.resize":
		return "Resize an image artifact to requested dimensions."
	default:
		return "Execute the observed provider capability " + capabilityID + "."
	}
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

const testPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAUAAAAFCAIAAAACDbGyAAAAFElEQVR4nGM8ceIEA27AhEduBEsDABUMAYuJ1HWoAAAAAElFTkSuQmCC"

func mustDecodeString(value string) string {
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		panic(err)
	}
	return string(decoded)
}
