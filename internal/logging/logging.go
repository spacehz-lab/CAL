package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spacehz-lab/cal/internal/config"
)

// EnvLevel overrides the configured logging level for one run.
const EnvLevel = "CAL_LOG_LEVEL"

const (
	defaultName = "cal"
	logExt      = ".log"
)

var logDir = defaultDir

// Options controls logging initialization for one named subject.
type Options struct {
	Name   string
	Config config.LoggingConfig
	Err    io.Writer
	Env    []string
}

// Result describes the effective logging setup.
type Result struct {
	Level       config.LoggingLevel
	FileEnabled bool
	FilePath    string
	ConfigError error
	FileError   error
}

// Configure installs the slog handler for one named subject.
func Configure(opts *Options) *Result {
	if opts == nil {
		opts = &Options{}
	}
	errWriter := opts.Err
	if errWriter == nil {
		errWriter = io.Discard
	}

	cfg, result := usableConfig(opts.Config)
	level := cfg.Level
	if env := strings.TrimSpace(envValue(optionsEnv(opts), EnvLevel)); env != "" {
		if normalized, err := config.NormalizeLoggingLevel(config.LoggingLevel(env)); err == nil {
			level = normalized
		} else {
			level = config.LoggingLevelError
			result.ConfigError = err
		}
	}
	result.Level = level

	writer := errWriter
	if cfg.File.FileEnabled() {
		result.FileEnabled = true
		path, err := Path(opts.Name)
		if err != nil {
			result.FileError = err
		} else if fileWriter, err := newRotatingWriter(path, cfg.File.MaxBytes, cfg.File.MaxFiles); err != nil {
			result.FilePath = path
			result.FileError = err
		} else {
			result.FilePath = path
			writer = io.MultiWriter(errWriter, fileWriter)
		}
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level: slogLevel(level),
	})))
	return result
}

// Path returns the platform log file path for one stable logging name.
func Path(name string) (string, error) {
	dir, err := logDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, cleanName(name)+logExt), nil
}

func usableConfig(cfg config.LoggingConfig) (config.LoggingConfig, *Result) {
	result := &Result{}
	if err := cfg.Validate(); err != nil {
		result.ConfigError = err
		cfg = config.Default().Logging
	}
	return cfg, result
}

func optionsEnv(opts *Options) []string {
	if opts.Env != nil {
		return opts.Env
	}
	return os.Environ()
}

func envValue(env []string, key string) string {
	prefix := key + "="
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			return item[len(prefix):]
		}
	}
	return ""
}

func cleanName(name string) string {
	name = strings.TrimSpace(filepath.Base(name))
	if name == "" || name == "." || name == string(filepath.Separator) {
		return defaultName
	}
	return name
}

func slogLevel(level config.LoggingLevel) slog.Level {
	switch level {
	case config.LoggingLevelDebug:
		return slog.LevelDebug
	case config.LoggingLevelInfo:
		return slog.LevelInfo
	case config.LoggingLevelWarn:
		return slog.LevelWarn
	default:
		return slog.LevelError
	}
}
