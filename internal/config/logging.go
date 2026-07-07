package config

import (
	"fmt"
	"strings"
)

// LoggingLevel identifies a durable logging level.
type LoggingLevel string

const (
	LoggingLevelDebug LoggingLevel = "debug"
	LoggingLevelInfo  LoggingLevel = "info"
	LoggingLevelWarn  LoggingLevel = "warn"
	LoggingLevelError LoggingLevel = "error"

	loggingLevelWarning = "warning"

	DefaultLogMaxBytes int64 = 10 * 1024 * 1024
	DefaultLogMaxFiles       = 5
)

// LoggingConfig contains durable local diagnostic logging settings.
type LoggingConfig struct {
	Level LoggingLevel      `json:"level,omitempty"`
	File  LoggingFileConfig `json:"file,omitempty"`
}

// LoggingFileConfig controls local rolling log files.
type LoggingFileConfig struct {
	Enabled  *bool `json:"enabled,omitempty"`
	MaxBytes int64 `json:"max_bytes,omitempty"`
	MaxFiles int   `json:"max_files,omitempty"`
}

// Validate checks logging configuration.
func (cfg *LoggingConfig) Validate() error {
	if _, err := NormalizeLoggingLevel(cfg.Level); err != nil {
		return err
	}
	if cfg.File.MaxBytes <= 0 {
		return fmt.Errorf("logging file max_bytes must be positive")
	}
	if cfg.File.MaxFiles <= 0 {
		return fmt.Errorf("logging file max_files must be positive")
	}
	return nil
}

// FileEnabled reports whether file logging is enabled after defaults.
func (cfg *LoggingFileConfig) FileEnabled() bool {
	return cfg.Enabled == nil || *cfg.Enabled
}

// NormalizeLoggingLevel validates and normalizes a configured log level.
func NormalizeLoggingLevel(level LoggingLevel) (LoggingLevel, error) {
	switch strings.ToLower(strings.TrimSpace(string(level))) {
	case string(LoggingLevelDebug):
		return LoggingLevelDebug, nil
	case "", string(LoggingLevelInfo):
		return LoggingLevelInfo, nil
	case string(LoggingLevelWarn), loggingLevelWarning:
		return LoggingLevelWarn, nil
	case string(LoggingLevelError):
		return LoggingLevelError, nil
	default:
		return "", fmt.Errorf("unsupported logging level %q", level)
	}
}

func (cfg LoggingConfig) withDefaults() LoggingConfig {
	defaults := defaultLogging()
	if strings.TrimSpace(string(cfg.Level)) == "" {
		cfg.Level = defaults.Level
	}
	level, err := NormalizeLoggingLevel(cfg.Level)
	if err == nil {
		cfg.Level = level
	}
	cfg.File = cfg.File.withDefaults(defaults.File)
	return cfg
}

func (cfg LoggingFileConfig) withDefaults(defaults LoggingFileConfig) LoggingFileConfig {
	if cfg.Enabled == nil {
		cfg.Enabled = defaults.Enabled
	}
	if cfg.MaxBytes == 0 {
		cfg.MaxBytes = defaults.MaxBytes
	}
	if cfg.MaxFiles == 0 {
		cfg.MaxFiles = defaults.MaxFiles
	}
	return cfg
}

func defaultLogging() LoggingConfig {
	enabled := true
	return LoggingConfig{
		Level: LoggingLevelInfo,
		File: LoggingFileConfig{
			Enabled:  &enabled,
			MaxBytes: DefaultLogMaxBytes,
			MaxFiles: DefaultLogMaxFiles,
		},
	}
}
