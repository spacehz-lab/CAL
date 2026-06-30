package runtime

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spacehz-lab/cal/internal/core"
)

// ExecutionResult captures one provider execution attempt.
type ExecutionResult struct {
	Stdout   string
	Stderr   string
	Output   string
	ExitCode int
}

// Supports reports whether the runner can execute this execution kind.
func (runner Runner) Supports(kind core.ExecutionKind) bool {
	switch kind {
	case core.ExecutionKindCLI:
		return true
	default:
		return false
	}
}

// Execute runs one supported execution plan against a provider.
func (runner Runner) Execute(ctx context.Context, provider core.Provider, execution core.Execution, inputs map[string]any) (ExecutionResult, error) {
	started := time.Now()
	switch execution.Kind {
	case core.ExecutionKindCLI:
		result, err := executeCLI(ctx, provider, execution, inputs)
		if err != nil {
			runner.logExecutionFailed(provider, execution, result, "execute", started)
			return result, err
		}
		runner.logExecutionCompleted(provider, execution, result, started)
		return result, nil
	default:
		runner.logExecutionFailed(provider, execution, ExecutionResult{}, "kind", started)
		return ExecutionResult{}, fmt.Errorf("execution kind %q is not supported", execution.Kind)
	}
}

func executeCLI(ctx context.Context, provider core.Provider, execution core.Execution, inputs map[string]any) (ExecutionResult, error) {
	if provider.Kind != core.ProviderKindCLI {
		return ExecutionResult{}, fmt.Errorf("provider kind %q cannot run cli execution", provider.Kind)
	}
	args, err := renderArgs(execution, inputs)
	if err != nil {
		return ExecutionResult{}, err
	}
	cmd := exec.CommandContext(ctx, provider.Path, args...)
	stdout, stderr, err := runCLICommand(cmd, execution, inputs)
	result := ExecutionResult{Stdout: stdout, Stderr: stderr, Output: stdout + stderr}
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}
	if err != nil {
		return result, fmt.Errorf("execute cli: %w", err)
	}
	return result, nil
}

func runCLICommand(cmd *exec.Cmd, execution core.Execution, inputs map[string]any) (string, string, error) {
	stdoutPathInput, ok, err := stdoutPathInput(execution)
	if err != nil {
		return "", "", err
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stdout.String(), stderr.String(), err
	}
	if !ok {
		return stdout.String(), stderr.String(), nil
	}

	path, ok := inputs[stdoutPathInput].(string)
	if !ok || path == "" {
		return stdout.String(), stderr.String(), fmt.Errorf("stdout path input %q is required", stdoutPathInput)
	}
	if err := os.WriteFile(path, stdout.Bytes(), 0o644); err != nil {
		return stdout.String(), stderr.String(), fmt.Errorf("write stdout target: %w", err)
	}
	return stdout.String(), stderr.String(), nil
}

func stdoutPathInput(execution core.Execution) (string, bool, error) {
	value, ok := execution.Spec[core.ExecutionSpecStdoutPathInput]
	if !ok {
		return "", false, nil
	}
	input, ok := value.(string)
	if !ok || input == "" {
		return "", false, fmt.Errorf("cli execution stdout path input must be a string")
	}
	return input, true, nil
}

func renderArgs(execution core.Execution, inputs map[string]any) ([]string, error) {
	args, err := executionArgs(execution)
	if err != nil {
		return nil, err
	}
	rendered := make([]string, len(args))
	for index, arg := range args {
		rendered[index] = renderArg(arg, inputs)
		if strings.Contains(rendered[index], "{{") || strings.Contains(rendered[index], "}}") {
			return nil, fmt.Errorf("missing input for cli arg template %q", arg)
		}
	}
	return rendered, nil
}

func renderArg(arg string, inputs map[string]any) string {
	for key, value := range inputs {
		arg = strings.ReplaceAll(arg, "{{"+key+"}}", fmt.Sprint(value))
	}
	return arg
}

func stringSlice(value any) ([]string, error) {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...), nil
	case []any:
		args := make([]string, len(typed))
		for index, item := range typed {
			arg, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("cli execution args must be strings")
			}
			args[index] = arg
		}
		return args, nil
	default:
		return nil, fmt.Errorf("cli execution args must be a string array")
	}
}

func (runner Runner) logExecutionCompleted(provider core.Provider, execution core.Execution, result ExecutionResult, started time.Time) {
	slog.Info("runtime execution completed",
		"provider_id", provider.ID,
		"provider_kind", provider.Kind,
		"execution_kind", execution.Kind,
		"exit_code", result.ExitCode,
		"output_bytes", len(result.Output),
		"duration_ms", time.Since(started).Milliseconds(),
	)
}

func (runner Runner) logExecutionFailed(provider core.Provider, execution core.Execution, result ExecutionResult, stage string, started time.Time) {
	slog.Warn("runtime execution failed",
		"provider_id", provider.ID,
		"provider_kind", provider.Kind,
		"execution_kind", execution.Kind,
		"exit_code", result.ExitCode,
		"output_bytes", len(result.Output),
		"stage", stage,
		"duration_ms", time.Since(started).Milliseconds(),
	)
}
