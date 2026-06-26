package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

func TestExecuteCLIWithRenderedArgs(t *testing.T) {
	dir := t.TempDir()
	providerPath := filepath.Join(dir, "fake-cli")
	writeExecutable(t, providerPath, "#!/bin/sh\ncp \"$3\" \"$5\"\n")
	source := filepath.Join(dir, "input.txt")
	target := filepath.Join(dir, "output.txt")
	if err := os.WriteFile(source, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	_, err := NewRunner(DefaultRegistry()).Execute(context.Background(), core.Provider{
		ID:   "provider_fake",
		Kind: core.ProviderKindCLI,
		Path: providerPath,
	}, core.Execution{
		Kind: core.ExecutionKindCLI,
		Spec: map[string]any{core.ExecutionSpecArgs: []string{"export-pdf", "--source", "{{source}}", "--target", "{{target}}"}},
	}, map[string]any{"source": source, "target": target})
	if err != nil {
		t.Fatalf("NewRunner(DefaultRegistry()).Execute() error = %v", err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("target missing: %v", err)
	}
}

func TestExecuteCLIRejectsMissingTemplateInput(t *testing.T) {
	_, err := NewRunner(DefaultRegistry()).Execute(context.Background(), core.Provider{
		ID:   "provider_fake",
		Kind: core.ProviderKindCLI,
		Path: "/bin/echo",
	}, core.Execution{
		Kind: core.ExecutionKindCLI,
		Spec: map[string]any{core.ExecutionSpecArgs: []string{"{{missing}}"}},
	}, map[string]any{})
	if err == nil {
		t.Fatal("NewRunner(DefaultRegistry()).Execute() error = nil, want missing input error")
	}
}

func TestExecuteCLIWritesStdoutToInputPath(t *testing.T) {
	dir := t.TempDir()
	providerPath := filepath.Join(dir, "stdout-cli")
	writeExecutable(t, providerPath, "#!/bin/sh\nprintf 'generated pdf bytes'\n")
	target := filepath.Join(dir, "output.pdf")

	_, err := NewRunner(DefaultRegistry()).Execute(context.Background(), core.Provider{
		ID:   "provider_stdout",
		Kind: core.ProviderKindCLI,
		Path: providerPath,
	}, core.Execution{
		Kind: core.ExecutionKindCLI,
		Spec: map[string]any{
			core.ExecutionSpecArgs:            []string{},
			core.ExecutionSpecStdoutPathInput: "target",
		},
	}, map[string]any{"target": target})
	if err != nil {
		t.Fatalf("NewRunner(DefaultRegistry()).Execute() error = %v", err)
	}
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(content) != "generated pdf bytes" {
		t.Fatalf("target content = %q, want stdout bytes", content)
	}
}

func TestExecuteCLIAcceptsJSONDecodedArgs(t *testing.T) {
	dir := t.TempDir()
	providerPath := filepath.Join(dir, "fake-cli")
	writeExecutable(t, providerPath, "#!/bin/sh\nexit 0\n")
	_, err := NewRunner(DefaultRegistry()).Execute(context.Background(), core.Provider{
		ID:   "provider_fake",
		Kind: core.ProviderKindCLI,
		Path: providerPath,
	}, core.Execution{
		Kind: core.ExecutionKindCLI,
		Spec: map[string]any{core.ExecutionSpecArgs: []any{"run", "{{value}}"}},
	}, map[string]any{"value": "ok"})
	if err != nil {
		t.Fatalf("NewRunner(DefaultRegistry()).Execute() error = %v", err)
	}
}

func TestExecuteCLIRejectsInvalidArgsSpec(t *testing.T) {
	_, err := NewRunner(DefaultRegistry()).Execute(context.Background(), core.Provider{
		ID:   "provider_fake",
		Kind: core.ProviderKindCLI,
		Path: "/bin/echo",
	}, core.Execution{
		Kind: core.ExecutionKindCLI,
		Spec: map[string]any{core.ExecutionSpecArgs: []any{"run", 1}},
	}, map[string]any{})
	if err == nil {
		t.Fatal("NewRunner(DefaultRegistry()).Execute() error = nil, want invalid args error")
	}
}

func TestExecuteCLIRejectsInvalidStdoutPathInputSpec(t *testing.T) {
	_, err := NewRunner(DefaultRegistry()).Execute(context.Background(), core.Provider{
		ID:   "provider_fake",
		Kind: core.ProviderKindCLI,
		Path: "/bin/echo",
	}, core.Execution{
		Kind: core.ExecutionKindCLI,
		Spec: map[string]any{
			core.ExecutionSpecArgs:            []string{},
			core.ExecutionSpecStdoutPathInput: 1,
		},
	}, map[string]any{})
	if err == nil {
		t.Fatal("NewRunner(DefaultRegistry()).Execute() error = nil, want invalid stdout path input spec error")
	}
}

func TestExecuteCLIReportsCommandFailure(t *testing.T) {
	dir := t.TempDir()
	providerPath := filepath.Join(dir, "fail-cli")
	writeExecutable(t, providerPath, "#!/bin/sh\nexit 7\n")
	result, err := NewRunner(DefaultRegistry()).Execute(context.Background(), core.Provider{
		ID:   "provider_fake",
		Kind: core.ProviderKindCLI,
		Path: providerPath,
	}, core.Execution{
		Kind: core.ExecutionKindCLI,
		Spec: map[string]any{core.ExecutionSpecArgs: []string{"run"}},
	}, map[string]any{})
	if err == nil {
		t.Fatal("NewRunner(DefaultRegistry()).Execute() error = nil, want command failure")
	}
	if result.ExitCode != 7 {
		t.Fatalf("exit code = %d, want 7", result.ExitCode)
	}
}

func writeExecutable(t *testing.T, path, script string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}
}
