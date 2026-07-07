package cli

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

var usageCommandTimeout = 10 * time.Second

const (
	sourceHelp      = "help"
	sourceDashHelp  = "dash_help"
	sourceShortHelp = "short_help"
	sourceUsage     = "usage"
	sourceMan       = "man"

	minUsefulUsageBytes = 200
)

// UsageOutput is one local usage surface for a CLI provider.
type UsageOutput struct {
	Source string
	Text   string
}

// UsageOutputs collects bounded local usage surfaces for a CLI provider.
func UsageOutputs(ctx context.Context, path string) ([]UsageOutput, error) {
	specs := []usageSpec{
		{source: sourceHelp, args: []string{"--help"}},
		{source: sourceDashHelp, args: []string{"-help"}},
		{source: sourceShortHelp, args: []string{"-h"}},
		{source: sourceUsage},
	}

	var firstErr error
	for _, spec := range specs {
		output, err := runUsageCommand(ctx, path, spec)
		if err == nil {
			return []UsageOutput{output}, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}

	output, err := usageFallback(ctx, path)
	if err == nil && isUsefulUsage(output.Text) {
		return []UsageOutput{output}, nil
	}
	if firstErr != nil {
		return nil, fmt.Errorf("observe cli usage: %w", firstErr)
	}
	if err != nil {
		return nil, fmt.Errorf("observe cli usage: %w", err)
	}
	return nil, fmt.Errorf("observe cli usage: no useful output")
}

type usageSpec struct {
	source string
	args   []string
}

func runUsageCommand(ctx context.Context, path string, spec usageSpec) (UsageOutput, error) {
	text, err := commandOutput(ctx, path, true, spec.args...)
	if err != nil {
		return UsageOutput{}, err
	}
	if !isUsefulUsage(text) {
		return UsageOutput{}, fmt.Errorf("command returned low-signal output")
	}
	return UsageOutput{Source: spec.source, Text: text}, nil
}

func commandOutput(ctx context.Context, path string, allowNonzeroOutput bool, args ...string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	runCtx, cancel := context.WithTimeout(ctx, usageCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, path, args...)
	out, err := cmd.CombinedOutput()
	text := string(out)
	if runCtx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("command timed out after %s", usageCommandTimeout)
	}
	if runCtx.Err() != nil {
		return "", runCtx.Err()
	}
	if err != nil {
		if allowNonzeroOutput && strings.TrimSpace(text) != "" {
			return text, nil
		}
		return "", err
	}
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("command returned empty output")
	}
	return text, nil
}

func isUsefulUsage(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	lower := strings.ToLower(trimmed)
	if hasOptionError(lower) {
		return hasUsageHeader(lower)
	}
	if hasUsageHeader(lower) || hasOptionLine(lower) {
		return true
	}
	return len(trimmed) >= minUsefulUsageBytes
}

func hasUsageHeader(lower string) bool {
	return strings.Contains(lower, "usage:") ||
		strings.Contains(lower, "options:") ||
		strings.Contains(lower, "command options") ||
		strings.Contains(lower, "commands:") ||
		strings.Contains(lower, "cal_capability") ||
		strings.Contains(lower, "cal_command")
}

func hasOptionLine(lower string) bool {
	for _, line := range strings.Split(lower, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "-") {
			return true
		}
	}
	return false
}

func hasOptionError(lower string) bool {
	return strings.Contains(lower, "unrecognized option") ||
		strings.Contains(lower, "unknown option") ||
		strings.Contains(lower, "invalid option") ||
		strings.Contains(lower, "illegal option") ||
		strings.Contains(lower, "bad option")
}
