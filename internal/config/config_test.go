package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureWritesMissingConfig(t *testing.T) {
	home := t.TempDir()
	configFile := New(home)
	cfg, err := configFile.Ensure()
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if cfg.Logging.Level != LogLevelInfo || !cfg.Logging.File.FileEnabled() {
		t.Fatalf("logging = %#v, want default file logging", cfg.Logging)
	}
	if _, err := os.Stat(filepath.Join(home, fileName)); err != nil {
		t.Fatalf("default config was not written: %v", err)
	}
}

func TestLoadConfig(t *testing.T) {
	home := t.TempDir()
	configFile := New(home)
	configPath := filepath.Join(home, fileName)
	if err := os.WriteFile(configPath, []byte(`{
  "llm": {
    "api": "chat_completions",
    "base_url": "https://api.example.test/v1",
    "model": "test-model",
    "api_key_ref": "env:CAL_TEST_LLM_API_KEY"
  },
  "logging": {
    "level": "warn",
    "file": {
      "enabled": false
    }
  }
}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := configFile.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.LLM == nil || cfg.LLM.API != LLMAPIChatCompletions || cfg.LLM.APIKeyRef != "env:CAL_TEST_LLM_API_KEY" {
		t.Fatalf("llm = %#v, want durable non-secret settings", cfg.LLM)
	}
	if cfg.Logging.Level != LogLevelWarn || cfg.Logging.File.FileEnabled() || cfg.Logging.File.MaxBytes == 0 || cfg.Logging.File.MaxFiles == 0 {
		t.Fatalf("logging = %#v, want configured level with default rotation", cfg.Logging)
	}
}

func TestResetOverwritesConfigWithDefaults(t *testing.T) {
	home := t.TempDir()
	configFile := New(home)
	configPath := filepath.Join(home, fileName)
	if err := os.WriteFile(configPath, []byte(`{"logging":{"level":"warn"}}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := configFile.reset()
	if err != nil {
		t.Fatalf("reset() error = %v", err)
	}
	loaded, err := configFile.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Logging.Level != cfg.Logging.Level {
		t.Fatalf("loaded logging = %#v, want %#v", loaded.Logging, cfg.Logging)
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	home := t.TempDir()
	configFile := New(home)
	configPath := filepath.Join(home, fileName)
	if err := os.WriteFile(configPath, []byte(`{"unexpected": true}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := configFile.Load(); err == nil {
		t.Fatal("Load() error = nil, want unknown field error")
	}
}

func TestLoadRejectsRawLLMAPIKey(t *testing.T) {
	home := t.TempDir()
	configFile := New(home)
	configPath := filepath.Join(home, fileName)
	if err := os.WriteFile(configPath, []byte(`{
  "llm": {
    "api": "chat_completions",
    "model": "test-model",
    "api_key": "sk-test"
  }
}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := configFile.Load(); err == nil {
		t.Fatal("Load() error = nil, want raw api_key rejection")
	}
}
