package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spacehz-lab/cal/internal/execute"
	"github.com/spacehz-lab/cal/internal/model"
)

func TestRunCapturesStdoutStderrAndExitCode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-cli")
	writeExecutable(t, path, "#!/bin/sh\nprintf 'ok\\n'\nprintf 'warn\\n' >&2\nexit 0\n")

	result, err := NewRunner().Run(context.Background(), request(path, []string{"run"}, nil))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Outputs[execute.OutputStdout].Text != "ok\n" {
		t.Fatalf("stdout = %q, want ok", result.Outputs[execute.OutputStdout].Text)
	}
	if result.Outputs[execute.OutputStderr].Text != "warn\n" {
		t.Fatalf("stderr = %q, want warn", result.Outputs[execute.OutputStderr].Text)
	}
	if number := result.Outputs[execute.OutputExitCode].Number; number == nil || *number != 0 {
		t.Fatalf("exit code = %#v, want 0", number)
	}
}

func TestRunDoesNotTreatNonZeroExitCodeAsExecutionError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fail-cli")
	writeExecutable(t, path, "#!/bin/sh\nprintf 'failed\\n' >&2\nexit 7\n")

	result, err := NewRunner().Run(context.Background(), request(path, []string{"run"}, nil))
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if number := result.Outputs[execute.OutputExitCode].Number; number == nil || *number != 7 {
		t.Fatalf("exit code = %#v, want 7", number)
	}
	if result.Outputs[execute.OutputStderr].Text != "failed\n" {
		t.Fatalf("stderr = %q, want failed", result.Outputs[execute.OutputStderr].Text)
	}
}

func TestRunWritesStdoutTarget(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stdout-cli")
	target := filepath.Join(dir, "output.txt")
	writeExecutable(t, path, "#!/bin/sh\nprintf 'artifact bytes'\n")

	req := request(path, []string{}, map[string]any{"target": target})
	req.Execution.Spec[model.ExecutionSpecStdoutPathInput] = "target"
	result, err := NewRunner().Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(content) != "artifact bytes" {
		t.Fatalf("target content = %q, want artifact bytes", content)
	}
	if result.Outputs[execute.OutputTarget].Path != target {
		t.Fatalf("target output = %#v, want path %q", result.Outputs[execute.OutputTarget], target)
	}
}

func TestRunAllowsCommandOwnedTargetOutput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file-output-cli")
	target := filepath.Join(dir, "output.txt")
	writeExecutable(t, path, "#!/bin/sh\nprintf 'artifact bytes' > \"$1\"\n")

	req := request(path, []string{"{{target}}"}, map[string]any{"target": target})
	req.Execution.Spec[model.ExecutionSpecStdoutPathInput] = nil
	result, err := NewRunner().Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(content) != "artifact bytes" {
		t.Fatalf("target content = %q, want artifact bytes", content)
	}
	if _, ok := result.Outputs[execute.OutputTarget]; ok {
		t.Fatalf("target output = %#v, want no stdout-captured target output", result.Outputs[execute.OutputTarget])
	}
}

func TestRunReportsMissingStdoutTarget(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stdout-cli")
	writeExecutable(t, path, "#!/bin/sh\nprintf 'artifact bytes'\n")

	req := request(path, []string{}, nil)
	req.Execution.Spec[model.ExecutionSpecStdoutPathInput] = "target"
	result, err := NewRunner().Run(context.Background(), req)
	if !errors.Is(err, ErrMissingStdoutTarget) {
		t.Fatalf("Run() error = %v, want ErrMissingStdoutTarget", err)
	}
	if result == nil || result.Outputs[execute.OutputStdout].Text != "artifact bytes" {
		t.Fatalf("result = %#v, want partial stdout result", result)
	}
}

func TestRunPropagatesContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := NewRunner().Run(ctx, request("/bin/sleep", []string{"5"}, nil))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run() error = %v, want context deadline exceeded", err)
	}
}

func request(path string, args []string, inputs map[string]any) *execute.Request {
	return &execute.Request{
		Provider: &model.Provider{
			Kind: model.ProviderKindCLI,
			Path: path,
		},
		Execution: &model.Execution{
			Kind: model.ExecutionKindCLI,
			Spec: map[string]any{model.ExecutionSpecArgs: args},
		},
		Inputs: inputs,
	}
}

func writeExecutable(t *testing.T, path string, script string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}
}
