package logging

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/config"
)

func TestConfigureUsesPassedConfigLevel(t *testing.T) {
	logs := useTempLogDir(t)
	var stderr bytes.Buffer
	restoreSlog(t)
	cfg := defaultLoggingConfig()
	disableFile(&cfg)
	cfg.Level = config.LoggingLevelWarn

	result := Configure(&Options{Name: "calctl", Config: cfg, Err: &stderr, Env: []string{}})
	slog.Info("hidden info")
	slog.Warn("visible warn")

	if result.Level != config.LoggingLevelWarn || result.FileEnabled || result.FilePath != "" {
		t.Fatalf("result = %#v, want warn stderr-only logging", result)
	}
	if strings.Contains(stderr.String(), "hidden info") || !strings.Contains(stderr.String(), "visible warn") {
		t.Fatalf("stderr = %q, want warn filtering", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(logs, "calctl.log")); !os.IsNotExist(err) {
		t.Fatalf("log file exists err=%v, want no file", err)
	}
}

func TestConfigureEnvLevelOverridesConfig(t *testing.T) {
	var stderr bytes.Buffer
	restoreSlog(t)
	cfg := defaultLoggingConfig()
	disableFile(&cfg)
	cfg.Level = config.LoggingLevelError

	result := Configure(&Options{Name: "calctl", Config: cfg, Err: &stderr, Env: []string{EnvLevel + "=info"}})
	slog.Info("env info")

	if result.Level != config.LoggingLevelInfo || result.ConfigError != nil {
		t.Fatalf("result = %#v, want info level without config error", result)
	}
	if !strings.Contains(stderr.String(), "env info") {
		t.Fatalf("stderr = %q, want env-overridden info log", stderr.String())
	}
}

func TestConfigureInvalidEnvLevelFallsBackToError(t *testing.T) {
	var stderr bytes.Buffer
	restoreSlog(t)
	cfg := defaultLoggingConfig()
	disableFile(&cfg)

	result := Configure(&Options{Name: "calctl", Config: cfg, Err: &stderr, Env: []string{EnvLevel + "=trace"}})
	slog.Warn("hidden warn")
	slog.Error("visible error")

	if result.Level != config.LoggingLevelError || result.ConfigError == nil {
		t.Fatalf("result = %#v, want error fallback and config error", result)
	}
	if strings.Contains(stderr.String(), "hidden warn") || !strings.Contains(stderr.String(), "visible error") {
		t.Fatalf("stderr = %q, want error filtering", stderr.String())
	}
}

func TestConfigureFileEnabledWritesToFileAndErr(t *testing.T) {
	logs := useTempLogDir(t)
	var stderr bytes.Buffer
	restoreSlog(t)
	cfg := defaultLoggingConfig()
	cfg.Level = config.LoggingLevelInfo

	result := Configure(&Options{Name: "calctl", Config: cfg, Err: &stderr, Env: []string{}})
	slog.Info("file info")

	wantPath := filepath.Join(logs, "calctl.log")
	if result.Level != config.LoggingLevelInfo || !result.FileEnabled || result.FilePath != wantPath || result.FileError != nil {
		t.Fatalf("result = %#v, want file logging at %s", result, wantPath)
	}
	content, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if !strings.Contains(string(content), "file info") || !strings.Contains(stderr.String(), "file info") {
		t.Fatalf("file=%q stderr=%q, want mirrored log", content, stderr.String())
	}
}

func TestConfigureInvalidConfigFallsBackToConfigDefaults(t *testing.T) {
	useTempLogDir(t)
	var stderr bytes.Buffer
	restoreSlog(t)
	cfg := config.LoggingConfig{Level: "trace"}
	disableFile(&cfg)

	result := Configure(&Options{Name: "calctl", Config: cfg, Err: &stderr, Env: []string{}})
	slog.Info("fallback info")

	if result.Level != config.LoggingLevelInfo || result.ConfigError == nil {
		t.Fatalf("result = %#v, want default info with config error", result)
	}
	if !strings.Contains(stderr.String(), "fallback info") {
		t.Fatalf("stderr = %q, want fallback info log", stderr.String())
	}
}

func TestConfigureDoesNotReadConfigFile(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, config.FileName), []byte(`{"unexpected":true}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	var stderr bytes.Buffer
	restoreSlog(t)
	cfg := defaultLoggingConfig()
	disableFile(&cfg)

	result := Configure(&Options{Name: "calctl", Config: cfg, Err: &stderr, Env: []string{}})
	slog.Info("configured directly")

	if result.ConfigError != nil || !strings.Contains(stderr.String(), "configured directly") {
		t.Fatalf("result=%#v stderr=%q, want no config file read", result, stderr.String())
	}
}

func TestPathSanitizesName(t *testing.T) {
	logs := useTempLogDir(t)
	tests := []struct {
		name string
		want string
	}{
		{name: "", want: filepath.Join(logs, defaultName+logExt)},
		{name: ".", want: filepath.Join(logs, defaultName+logExt)},
		{name: string(filepath.Separator), want: filepath.Join(logs, defaultName+logExt)},
		{name: "../cald", want: filepath.Join(logs, "cald"+logExt)},
	}
	for _, test := range tests {
		got, err := Path(test.name)
		if err != nil {
			t.Fatalf("Path(%q) error = %v", test.name, err)
		}
		if got != test.want {
			t.Fatalf("Path(%q) = %q, want %q", test.name, got, test.want)
		}
	}
}

func defaultLoggingConfig() config.LoggingConfig {
	return config.Default().Logging
}

func disableFile(cfg *config.LoggingConfig) {
	enabled := false
	cfg.File.Enabled = &enabled
}

func restoreSlog(t *testing.T) {
	t.Helper()
	previous := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(previous)
	})
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
