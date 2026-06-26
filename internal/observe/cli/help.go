package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var commandTimeout = 10 * time.Second

const (
	minUsefulDocumentationBytes = 200

	logKeyProviderName = "provider_name"
	logKeySource       = "source"
	logKeyCode         = "code"
	logKeyOutputBytes  = "output_bytes"
	logKeyDurationMS   = "duration_ms"
)

// DocumentationOutput is one local documentation surface for a CLI provider.
type DocumentationOutput struct {
	Source string
	Text   string
}

// DocumentationOutputs collects bounded local documentation surfaces for a CLI provider.
func DocumentationOutputs(ctx context.Context, path string) ([]DocumentationOutput, error) {
	started := time.Now()
	providerName := filepath.Base(path)
	logDocumentation(ctx, slog.LevelInfo, "observe cli documentation started", providerName)
	specs := []struct {
		source string
		args   []string
	}{
		{source: "help", args: []string{"--help"}},
		{source: "help", args: []string{"-help"}},
		{source: "short_help", args: []string{"-h"}},
		{source: "usage"},
	}

	var firstErr error
	for _, spec := range specs {
		attemptStarted := time.Now()
		text, err := commandOutput(ctx, path, true, spec.args...)
		if err != nil {
			logDocumentation(ctx, slog.LevelInfo, "observe cli documentation attempt failed", providerName,
				logKeySource, spec.source,
				logKeyCode, documentationErrorCode(err),
				logKeyDurationMS, time.Since(attemptStarted).Milliseconds(),
			)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if !isUsefulDocumentation(text) {
			err := fmt.Errorf("command returned low-signal output")
			logDocumentation(ctx, slog.LevelInfo, "observe cli documentation attempt failed", providerName,
				logKeySource, spec.source,
				logKeyCode, documentationErrorCode(err),
				logKeyOutputBytes, len(text),
				logKeyDurationMS, time.Since(attemptStarted).Milliseconds(),
			)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		logDocumentation(ctx, slog.LevelInfo, "observe cli documentation completed", providerName,
			logKeySource, spec.source,
			logKeyOutputBytes, len(text),
			logKeyDurationMS, time.Since(started).Milliseconds(),
		)
		return []DocumentationOutput{{Source: spec.source, Text: text}}, nil
	}

	fallbackStarted := time.Now()
	if output, err := documentationFallback(ctx, path); err == nil {
		logDocumentation(ctx, slog.LevelInfo, "observe cli documentation completed", providerName,
			logKeySource, output.Source,
			logKeyOutputBytes, len(output.Text),
			logKeyDurationMS, time.Since(started).Milliseconds(),
		)
		return []DocumentationOutput{output}, nil
	} else {
		logDocumentation(ctx, slog.LevelInfo, "observe cli documentation attempt failed", providerName,
			logKeySource, documentationFallbackSource(),
			logKeyCode, documentationErrorCode(err),
			logKeyDurationMS, time.Since(fallbackStarted).Milliseconds(),
		)
	}
	if firstErr != nil {
		logDocumentation(ctx, slog.LevelWarn, "observe cli documentation failed", providerName,
			logKeyCode, documentationErrorCode(firstErr),
			logKeyDurationMS, time.Since(started).Milliseconds(),
		)
		return nil, fmt.Errorf("observe cli documentation: %w", firstErr)
	}
	logDocumentation(ctx, slog.LevelWarn, "observe cli documentation failed", providerName,
		logKeyCode, "unavailable",
		logKeyDurationMS, time.Since(started).Milliseconds(),
	)
	return nil, nil
}

func logDocumentation(ctx context.Context, level slog.Level, message, providerName string, attrs ...any) {
	args := make([]any, 0, 2+len(attrs))
	args = append(args, logKeyProviderName, providerName)
	args = append(args, attrs...)
	slog.Log(ctx, level, message, args...)
}

func commandOutput(ctx context.Context, path string, allowNonzeroOutput bool, args ...string) (string, error) {
	runCtx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, path, args...)
	out, err := cmd.CombinedOutput()
	text := string(out)
	if runCtx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("command timed out after %s", commandTimeout)
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

func isUsefulDocumentation(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	lower := strings.ToLower(trimmed)
	if hasOptionError(lower) {
		return hasDocumentationHeader(lower)
	}
	if hasDocumentationHeader(lower) || hasOptionLine(lower) {
		return true
	}
	return len(trimmed) >= minUsefulDocumentationBytes
}

func hasDocumentationHeader(lower string) bool {
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

func documentationErrorCode(err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	switch {
	case strings.Contains(message, "timed out"):
		return "timeout"
	case strings.Contains(message, "empty"):
		return "empty_output"
	case strings.Contains(message, "low-signal"):
		return "low_signal_output"
	default:
		return "command_failed"
	}
}
