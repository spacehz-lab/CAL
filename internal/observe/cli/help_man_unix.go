//go:build !windows

package cli

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

func documentationFallback(ctx context.Context, path string) (DocumentationOutput, error) {
	text, err := manOutput(ctx, path)
	if err != nil {
		return DocumentationOutput{}, err
	}
	return DocumentationOutput{Source: documentationFallbackSource(), Text: text}, nil
}

func documentationFallbackSource() string {
	return "man"
}

func manOutput(ctx context.Context, path string) (string, error) {
	name := filepath.Base(path)
	if strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("man name is required")
	}
	resolved, err := exec.LookPath(name)
	if err != nil {
		return "", err
	}
	resolvedAbs, err := filepath.Abs(resolved)
	if err != nil {
		return "", err
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if filepath.Clean(resolvedAbs) != filepath.Clean(pathAbs) {
		return "", fmt.Errorf("man name %q resolves to %q, not provider path %q", name, resolvedAbs, pathAbs)
	}
	runCtx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, "man", name)
	cmd.Env = append(cmd.Environ(), "MANPAGER=cat", "PAGER=cat")
	out, err := cmd.CombinedOutput()
	if runCtx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("man timed out after %s", commandTimeout)
	}
	if err != nil {
		return "", err
	}
	text := string(out)
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("man returned empty output")
	}
	return text, nil
}
