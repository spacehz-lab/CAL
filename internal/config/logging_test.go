package config

import "testing"

func TestDefaultLoggingConfig(t *testing.T) {
	cfg := defaultLogging()
	if cfg.Level != LoggingLevelInfo {
		t.Fatalf("Level = %q, want info", cfg.Level)
	}
	if !cfg.File.FileEnabled() {
		t.Fatal("FileEnabled() = false, want true")
	}
	if cfg.File.MaxBytes != DefaultLogMaxBytes || cfg.File.MaxFiles != DefaultLogMaxFiles {
		t.Fatalf("File = %#v, want default rotation", cfg.File)
	}
}

func TestLoggingConfigAppliesDefaults(t *testing.T) {
	cfg := (&Config{}).WithDefaults()
	if cfg.Logging.Level != LoggingLevelInfo || !cfg.Logging.File.FileEnabled() || cfg.Logging.File.MaxBytes != DefaultLogMaxBytes || cfg.Logging.File.MaxFiles != DefaultLogMaxFiles {
		t.Fatalf("Logging = %#v, want defaults", cfg.Logging)
	}
}

func TestNormalizeLoggingLevel(t *testing.T) {
	tests := []struct {
		value LoggingLevel
		want  LoggingLevel
	}{
		{value: LoggingLevelDebug, want: LoggingLevelDebug},
		{value: " info ", want: LoggingLevelInfo},
		{value: loggingLevelWarning, want: LoggingLevelWarn},
		{value: LoggingLevelError, want: LoggingLevelError},
	}
	for _, test := range tests {
		got, err := NormalizeLoggingLevel(test.value)
		if err != nil {
			t.Fatalf("NormalizeLoggingLevel(%q) error = %v", test.value, err)
		}
		if got != test.want {
			t.Fatalf("NormalizeLoggingLevel(%q) = %q, want %q", test.value, got, test.want)
		}
	}
}

func TestLoggingConfigRejectsInvalidLevel(t *testing.T) {
	cfg := LoggingConfig{Level: "trace"}.withDefaults()
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want invalid level error")
	}
}

func TestLoggingConfigRejectsNegativeRotation(t *testing.T) {
	tests := []LoggingConfig{
		{File: LoggingFileConfig{MaxBytes: -1}},
		{File: LoggingFileConfig{MaxFiles: -1}},
	}
	for _, cfg := range tests {
		cfg = cfg.withDefaults()
		if err := cfg.Validate(); err == nil {
			t.Fatal("Validate() error = nil, want rotation error")
		}
	}
}
