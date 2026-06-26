package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spacehz-lab/cal/internal/calpath"
	"github.com/spacehz-lab/cal/internal/config"
)

// EnvLogLevel overrides the configured logging level for one process.
const EnvLogLevel = "CAL_LOG_LEVEL"

var logDir = defaultLogDir

// Options controls process logging initialization.
type Options struct {
	Process string
	Home    string
	Err     io.Writer
}

// Result describes the effective logging setup.
type Result struct {
	Level       string
	FileEnabled bool
	FilePath    string
	ConfigError error
	FileError   error
}

// Configure installs the process slog handler.
func Configure(opts Options) Result {
	if opts.Err == nil {
		opts.Err = io.Discard
	}
	cfg, result := loadConfig(opts.Home)
	level := cfg.Level
	if env := strings.TrimSpace(os.Getenv(EnvLogLevel)); env != "" {
		if normalized, err := config.NormalizeLogLevel(env); err == nil {
			level = normalized
		} else {
			level = config.LogLevelError
			result.ConfigError = err
		}
	}
	result.Level = level
	w := opts.Err
	if cfg.File.FileEnabled() {
		result.FileEnabled = true
		path, err := logFilePath(opts.Process)
		if err != nil {
			result.FileError = err
		} else if fileWriter, err := newRotatingWriter(path, cfg.File.MaxBytes, cfg.File.MaxFiles); err != nil {
			result.FilePath = path
			result.FileError = err
		} else {
			result.FilePath = path
			w = io.MultiWriter(opts.Err, fileWriter)
		}
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: slogLevel(level),
	})))
	return result
}

func loadConfig(home string) (config.Logging, Result) {
	var result Result
	if strings.TrimSpace(home) == "" {
		resolved, err := calpath.HomeDir()
		if err != nil {
			result.ConfigError = err
			return config.DefaultLogging(), result
		}
		home = resolved
	}
	cfg, err := config.New(home).Load()
	if err != nil {
		result.ConfigError = err
		return config.DefaultLogging(), result
	}
	return cfg.Logging, result
}

func logFilePath(process string) (string, error) {
	dir, err := logDir()
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(filepath.Base(process))
	if name == "." || name == string(filepath.Separator) || name == "" {
		name = "cal"
	}
	return filepath.Join(dir, name+".log"), nil
}

// ProcessLogPath returns the platform log path for one CAL process name.
func ProcessLogPath(process string) (string, error) {
	return logFilePath(process)
}

func slogLevel(level string) slog.Level {
	switch level {
	case config.LogLevelDebug:
		return slog.LevelDebug
	case config.LogLevelInfo:
		return slog.LevelInfo
	case config.LogLevelWarn:
		return slog.LevelWarn
	default:
		return slog.LevelError
	}
}
