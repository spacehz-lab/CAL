package config

import "testing"

func TestDefaultLoggingEnablesInfoFileLogging(t *testing.T) {
	cfg := DefaultLogging()
	if cfg.Level != LogLevelInfo {
		t.Fatalf("level = %q, want info", cfg.Level)
	}
	if !cfg.File.FileEnabled() {
		t.Fatal("file enabled = false, want true")
	}
	if cfg.File.MaxBytes != defaultLogMaxBytes || cfg.File.MaxFiles != defaultLogMaxFiles {
		t.Fatalf("file = %#v, want default rotation", cfg.File)
	}
}

func TestLoggingDefaultsMissingFields(t *testing.T) {
	cfg := Config{}
	if err := cfg.applyDefaults(); err != nil {
		t.Fatalf("applyDefaults() error = %v", err)
	}
	if cfg.Logging.Level != LogLevelInfo || !cfg.Logging.File.FileEnabled() || cfg.Logging.File.MaxBytes == 0 || cfg.Logging.File.MaxFiles == 0 {
		t.Fatalf("logging = %#v, want defaults", cfg.Logging)
	}
}

func TestNormalizeLogLevel(t *testing.T) {
	tests := []struct {
		value string
		want  string
	}{
		{value: "debug", want: LogLevelDebug},
		{value: " info ", want: LogLevelInfo},
		{value: "warning", want: LogLevelWarn},
		{value: "error", want: LogLevelError},
	}
	for _, test := range tests {
		got, err := NormalizeLogLevel(test.value)
		if err != nil {
			t.Fatalf("NormalizeLogLevel(%q) error = %v", test.value, err)
		}
		if got != test.want {
			t.Fatalf("NormalizeLogLevel(%q) = %q, want %q", test.value, got, test.want)
		}
	}
}

func TestNormalizeLogLevelRejectsUnknown(t *testing.T) {
	if _, err := NormalizeLogLevel("trace"); err == nil {
		t.Fatal("NormalizeLogLevel() error = nil, want unknown level rejection")
	}
}
