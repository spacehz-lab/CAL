package config

import (
	"fmt"
	"strings"
)

const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"

	defaultLogMaxBytes int64 = 10 * 1024 * 1024
	defaultLogMaxFiles       = 5
)

// Logging contains durable local diagnostic logging settings.
type Logging struct {
	Level string      `json:"level,omitempty"`
	File  LoggingFile `json:"file,omitempty"`
}

// LoggingFile controls local rolling log files.
type LoggingFile struct {
	Enabled  *bool `json:"enabled,omitempty"`
	MaxBytes int64 `json:"max_bytes,omitempty"`
	MaxFiles int   `json:"max_files,omitempty"`
}

// DefaultLogging returns product defaults for local diagnostic logging.
func DefaultLogging() Logging {
	enabled := true
	return Logging{
		Level: LogLevelInfo,
		File: LoggingFile{
			Enabled:  &enabled,
			MaxBytes: defaultLogMaxBytes,
			MaxFiles: defaultLogMaxFiles,
		},
	}
}

// FileEnabled reports whether file logging is enabled after defaults.
func (cfg LoggingFile) FileEnabled() bool {
	return cfg.Enabled == nil || *cfg.Enabled
}

func (cfg *Config) applyDefaults() error {
	logging, err := cfg.Logging.withDefaults()
	if err != nil {
		return err
	}
	cfg.Logging = logging
	return nil
}

func (cfg Logging) withDefaults() (Logging, error) {
	defaults := DefaultLogging()
	if strings.TrimSpace(cfg.Level) == "" {
		cfg.Level = defaults.Level
	}
	level, err := NormalizeLogLevel(cfg.Level)
	if err != nil {
		return Logging{}, err
	}
	cfg.Level = level
	file, err := cfg.File.withDefaults(defaults.File)
	if err != nil {
		return Logging{}, err
	}
	cfg.File = file
	return cfg, nil
}

func (cfg LoggingFile) withDefaults(defaults LoggingFile) (LoggingFile, error) {
	if cfg.Enabled == nil {
		cfg.Enabled = defaults.Enabled
	}
	if cfg.MaxBytes == 0 {
		cfg.MaxBytes = defaults.MaxBytes
	}
	if cfg.MaxFiles == 0 {
		cfg.MaxFiles = defaults.MaxFiles
	}
	if cfg.MaxBytes < 0 {
		return LoggingFile{}, fmt.Errorf("logging file max_bytes must be positive")
	}
	if cfg.MaxFiles < 0 {
		return LoggingFile{}, fmt.Errorf("logging file max_files must be positive")
	}
	return cfg, nil
}

// NormalizeLogLevel validates and normalizes a configured log level.
func NormalizeLogLevel(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case LogLevelDebug:
		return LogLevelDebug, nil
	case LogLevelInfo:
		return LogLevelInfo, nil
	case LogLevelWarn, "warning":
		return LogLevelWarn, nil
	case LogLevelError:
		return LogLevelError, nil
	default:
		return "", fmt.Errorf("unsupported logging level %q", value)
	}
}
