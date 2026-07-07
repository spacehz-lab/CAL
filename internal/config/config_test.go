package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingConfigReturnsDefaultsWithoutWriting(t *testing.T) {
	file := NewFile(t.TempDir())

	cfg, err := file.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Logging.Level != LoggingLevelInfo || !cfg.Logging.File.FileEnabled() {
		t.Fatalf("Load() = %#v, want logging defaults", cfg)
	}
	if _, err := os.Stat(file.Path()); !os.IsNotExist(err) {
		t.Fatalf("config file exists after Load(): %v", err)
	}
}

func TestEnsureCreatesDefaultConfig(t *testing.T) {
	file := NewFile(t.TempDir())

	cfg, err := file.Ensure()
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if cfg.Logging.Level != LoggingLevelInfo {
		t.Fatalf("Ensure() logging level = %q, want info", cfg.Logging.Level)
	}
	content, err := os.ReadFile(file.Path())
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.Contains(string(content), `"llm"`) {
		t.Fatalf("default config contains llm section: %s", content)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	file := NewFile(t.TempDir())
	enabled := false
	cfg := &Config{
		LLM: &LLMConfig{
			API:       LLMAPIChatCompletions,
			BaseURL:   " https://api.example.test/v1 ",
			Model:     " test-model ",
			APIKeyRef: " env:CAL_TEST_LLM_API_KEY ",
		},
		Logging: LoggingConfig{
			Level: loggingLevelWarning,
			File: LoggingFileConfig{
				Enabled:  &enabled,
				MaxBytes: 2048,
				MaxFiles: 7,
			},
		},
	}

	if err := file.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	loaded, err := file.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.LLM == nil || loaded.LLM.API != LLMAPIChatCompletions || loaded.LLM.BaseURL != "https://api.example.test/v1" || loaded.LLM.Model != "test-model" || loaded.LLM.APIKeyRef != "env:CAL_TEST_LLM_API_KEY" {
		t.Fatalf("loaded llm = %#v, want trimmed durable settings", loaded.LLM)
	}
	if loaded.Logging.Level != LoggingLevelWarn || loaded.Logging.File.FileEnabled() || loaded.Logging.File.MaxBytes != 2048 || loaded.Logging.File.MaxFiles != 7 {
		t.Fatalf("loaded logging = %#v, want configured logging", loaded.Logging)
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	file := writeConfig(t, `{"unexpected": true}`)

	if _, err := file.Load(); err == nil {
		t.Fatal("Load() error = nil, want unknown field error")
	}
}

func TestLoadRejectsRawLLMAPIKey(t *testing.T) {
	file := writeConfig(t, `{"llm":{"api_key":"sk-test"}}`)

	if _, err := file.Load(); err == nil {
		t.Fatal("Load() error = nil, want raw api key rejection")
	}
}

func TestLoadKeepsAPIKeyRefUnresolved(t *testing.T) {
	file := writeConfig(t, `{"llm":{"api_key_ref":"env:CAL_TEST_LLM_API_KEY"}}`)

	cfg, err := file.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.LLM == nil || cfg.LLM.APIKeyRef != "env:CAL_TEST_LLM_API_KEY" {
		t.Fatalf("llm = %#v, want unresolved api key ref", cfg.LLM)
	}
}

func writeConfig(t *testing.T, content string) *File {
	t.Helper()
	home := t.TempDir()
	path := filepath.Join(home, FileName)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return NewFile(home)
}
