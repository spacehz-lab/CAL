package execute

import (
	"context"

	"github.com/spacehz-lab/cal/internal/model"
)

const (
	OutputStdout     OutputName = "stdout"
	OutputStderr     OutputName = "stderr"
	OutputExitCode   OutputName = "exit_code"
	OutputTarget     OutputName = "target"
	OutputText       OutputName = "text"
	OutputScreenshot OutputName = "screenshot"
	OutputURL        OutputName = "url"
	OutputDOMText    OutputName = "dom_text"
)

const (
	OutputKindText   OutputKind = "text"
	OutputKindNumber OutputKind = "number"
	OutputKindFile   OutputKind = "file"
	OutputKindLink   OutputKind = "link"
)

// Executor runs one provider-specific execution request.
type Executor interface {
	Run(context.Context, *Request) (*Result, error)
}

// Request provides one provider execution run input.
type Request struct {
	Provider  *model.Provider
	Execution *model.Execution
	Inputs    map[string]any
}

// Result describes outputs produced by one execution attempt.
type Result struct {
	Outputs Outputs
}

// Outputs maps stable semantic output names to typed output values.
type Outputs map[OutputName]Output

// OutputName identifies a stable execution output.
type OutputName string

// Output describes one typed execution output.
type Output struct {
	Kind   OutputKind
	Text   string
	Number *int
	Path   string
	MIME   string
}

// OutputKind identifies the output value shape.
type OutputKind string
