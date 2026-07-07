package model

const (
	// ExecutionSpecArgs names the argv array for CLI execution.
	ExecutionSpecArgs = "args"
	// ExecutionSpecStdoutPathInput names the input key whose path receives stdout.
	ExecutionSpecStdoutPathInput = "stdout_path_input"
)

// ExecutionKind identifies the supported binding execution type.
type ExecutionKind string

const (
	// ExecutionKindCLI runs a command-line execution plan.
	ExecutionKindCLI ExecutionKind = "cli"
	// ExecutionKindMenu runs a menu-driven execution plan.
	ExecutionKindMenu ExecutionKind = "menu"
	// ExecutionKindAXAction runs an accessibility action execution plan.
	ExecutionKindAXAction ExecutionKind = "ax_action"
	// ExecutionKindURLOpen opens a URL as the execution plan.
	ExecutionKindURLOpen ExecutionKind = "url_open"
)

// Execution is the concrete provider-specific execution plan.
type Execution struct {
	Kind ExecutionKind  `json:"kind"`
	Spec map[string]any `json:"spec,omitempty"`
}
