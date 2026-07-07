package check

import "github.com/spacehz-lab/cal/internal/model"

// Request provides one deterministic verification run input.
type Request struct {
	Spec     *model.VerifySpec
	Inputs   map[string]any
	Stdout   string
	Stderr   string
	ExitCode int
}

// Result describes passed verification evidence and checked outputs.
type Result struct {
	Evidence []model.EvidenceRef
	Outputs  map[string]any
}
