package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spacehz-lab/cal/internal/execute"
)

func runCommand(ctx context.Context, req *execute.Request) (*execute.Result, error) {
	args, err := execute.RenderArgs(req.Execution, req.Inputs)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, strings.TrimSpace(req.Provider.Path), args...)
	stdout, stderr, exitCode, err := runProcess(cmd)
	result := resultFromProcess(stdout, stderr, exitCode)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return result, ctxErr
		}
		if exitCode == nil {
			return result, fmt.Errorf("execute cli: %w", err)
		}
	}
	if err := writeStdoutTarget(req, stdout, result); err != nil {
		return result, err
	}
	return result, nil
}

func runProcess(cmd *exec.Cmd) (string, string, *int, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if cmd.ProcessState == nil {
		return stdout.String(), stderr.String(), nil, err
	}
	exitCode := cmd.ProcessState.ExitCode()
	return stdout.String(), stderr.String(), &exitCode, err
}

func resultFromProcess(stdout string, stderr string, exitCode *int) *execute.Result {
	outputs := execute.Outputs{
		execute.OutputStdout: {
			Kind: execute.OutputKindText,
			Text: stdout,
		},
		execute.OutputStderr: {
			Kind: execute.OutputKindText,
			Text: stderr,
		},
		execute.OutputText: {
			Kind: execute.OutputKindText,
			Text: stdout,
		},
	}
	if exitCode != nil {
		outputs[execute.OutputExitCode] = execute.Output{
			Kind:   execute.OutputKindNumber,
			Number: exitCode,
		}
	}
	return &execute.Result{Outputs: outputs}
}

func writeStdoutTarget(req *execute.Request, stdout string, result *execute.Result) error {
	stdoutPathInput, ok, err := execute.StdoutPathInput(req.Execution)
	if err != nil || !ok {
		return err
	}
	path, ok := req.Inputs[stdoutPathInput].(string)
	if !ok || strings.TrimSpace(path) == "" {
		return fmt.Errorf("%w: %s", ErrMissingStdoutTarget, stdoutPathInput)
	}
	path = strings.TrimSpace(path)
	if err := os.WriteFile(path, []byte(stdout), 0o644); err != nil {
		return fmt.Errorf("write stdout target: %w", err)
	}
	result.Outputs[execute.OutputName(stdoutPathInput)] = execute.Output{
		Kind: execute.OutputKindFile,
		Path: path,
	}
	return nil
}
