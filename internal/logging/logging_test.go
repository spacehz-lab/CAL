package logging

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigureDefaultsToInfoFileLogging(t *testing.T) {
	t.Setenv(EnvLogLevel, "")
	home := t.TempDir()
	logs := useTempLogDir(t)
	var stderr bytes.Buffer
	previous := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(previous)
	})

	result := Configure(Options{Process: "calctl", Home: home, Err: &stderr})
	slog.Info("default info log", "case", "default")

	if result.Level != "info" || !result.FileEnabled || result.FilePath != filepath.Join(logs, "calctl.log") {
		t.Fatalf("result = %#v, want info file logging under temp dir", result)
	}
	content, err := os.ReadFile(result.FilePath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if !strings.Contains(string(content), "default info log") {
		t.Fatalf("log file = %q, want info log", content)
	}
	if !strings.Contains(stderr.String(), "default info log") {
		t.Fatalf("stderr = %q, want mirrored info log", stderr.String())
	}
}

func TestConfigureUsesConfigLevel(t *testing.T) {
	t.Setenv(EnvLogLevel, "")
	home := t.TempDir()
	writeLoggingConfig(t, home, `{"level":"warn","file":{"enabled":false}}`)
	var stderr bytes.Buffer
	previous := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(previous)
	})

	result := Configure(Options{Process: "calctl", Home: home, Err: &stderr})
	slog.Info("hidden info")
	slog.Warn("visible warn")

	if result.Level != "warn" || result.FileEnabled {
		t.Fatalf("result = %#v, want warn stderr-only logging", result)
	}
	if strings.Contains(stderr.String(), "hidden info") || !strings.Contains(stderr.String(), "visible warn") {
		t.Fatalf("stderr = %q, want config level filtering", stderr.String())
	}
}

func TestConfigureEnvLogLevelOverridesConfig(t *testing.T) {
	t.Setenv(EnvLogLevel, "info")
	home := t.TempDir()
	writeLoggingConfig(t, home, `{"level":"error","file":{"enabled":false}}`)
	var stderr bytes.Buffer
	previous := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(previous)
	})

	result := Configure(Options{Process: "calctl", Home: home, Err: &stderr})
	slog.Info("env info")

	if result.Level != "info" {
		t.Fatalf("level = %q, want env override info", result.Level)
	}
	if !strings.Contains(stderr.String(), "env info") {
		t.Fatalf("stderr = %q, want env-overridden info log", stderr.String())
	}
}

func TestConfigureFallsBackWhenConfigIsInvalid(t *testing.T) {
	t.Setenv(EnvLogLevel, "")
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "config.json"), []byte(`{"unexpected":true}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	logs := useTempLogDir(t)
	var stderr bytes.Buffer
	previous := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(previous)
	})

	result := Configure(Options{Process: "calctl", Home: home, Err: &stderr})
	slog.Info("fallback info")

	if result.ConfigError == nil || result.Level != "info" || result.FilePath != filepath.Join(logs, "calctl.log") {
		t.Fatalf("result = %#v, want fallback defaults with config error", result)
	}
	if !strings.Contains(stderr.String(), "fallback info") {
		t.Fatalf("stderr = %q, want fallback info log", stderr.String())
	}
}

func TestProcessLogPathUsesPlatformLogDir(t *testing.T) {
	logs := useTempLogDir(t)

	path, err := ProcessLogPath("cald-daemon")
	if err != nil {
		t.Fatalf("ProcessLogPath() error = %v", err)
	}
	want := filepath.Join(logs, "cald-daemon.log")
	if path != want {
		t.Fatalf("ProcessLogPath() = %q, want %q", path, want)
	}
}

func writeLoggingConfig(t *testing.T, home string, logging string) {
	t.Helper()
	content := `{"provider_sources": [{"kind": "path", "value": "PATH"}],"logging":` + logging + `}`
	if err := os.WriteFile(filepath.Join(home, "config.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func useTempLogDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	previous := logDir
	logDir = func() (string, error) {
		return dir, nil
	}
	t.Cleanup(func() {
		logDir = previous
	})
	return dir
}
