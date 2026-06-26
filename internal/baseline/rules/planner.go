package rules

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/proposal"
)

const (
	verifierFileParsePDF    = "file_parse_pdf"
	verifierImageDimensions = "image_dimensions_match"
)

type probePlanner struct{}

// NewProbePlanner builds the deterministic baseline probe planner.
func NewProbePlanner() proposal.ProbePlanner {
	return probePlanner{}
}

func (probePlanner) Plan(_ context.Context, request proposal.ProbePlanRequest) (proposal.ProbePlan, error) {
	if request.WorkDir == "" {
		return proposal.ProbePlan{}, fmt.Errorf("probe work directory is required")
	}
	switch request.Candidate.CapabilityID {
	case "document.export_pdf":
		return planDocumentExportPDFProbe(request.WorkDir)
	case "image.resize":
		return planImageResizeProbe(request.WorkDir)
	default:
		return proposal.ProbePlan{}, fmt.Errorf("capability %q does not have a deterministic probe plan", request.Candidate.CapabilityID)
	}
}

func planDocumentExportPDFProbe(dir string) (proposal.ProbePlan, error) {
	source := filepath.Join(dir, "input.txt")
	target := filepath.Join(dir, "output.pdf")
	if err := os.WriteFile(source, []byte("cal probe input\n"), 0o644); err != nil {
		return proposal.ProbePlan{}, fmt.Errorf("write probe input: %w", err)
	}
	return proposal.ProbePlan{
		Inputs:   map[string]any{"source": source, "target": target},
		Verifier: core.Verifier{ID: verifierFileParsePDF},
	}, nil
}

func planImageResizeProbe(dir string) (proposal.ProbePlan, error) {
	source := filepath.Join(dir, "input.png")
	target := filepath.Join(dir, "output.png")
	if err := writeProbePNG(source, 4, 4); err != nil {
		return proposal.ProbePlan{}, err
	}
	return proposal.ProbePlan{
		Inputs: map[string]any{
			"source": source,
			"target": target,
			"width":  12,
			"height": 8,
		},
		Verifier: core.Verifier{ID: verifierImageDimensions},
	}, nil
}

func writeProbePNG(path string, width, height int) error {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 0x33, G: 0x99, B: 0xcc, A: 0xff})
		}
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create probe PNG: %w", err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		return fmt.Errorf("encode probe PNG: %w", err)
	}
	return nil
}
