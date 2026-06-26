package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDefaultConfigUsesPlatformProviderPaths(t *testing.T) {
	cfg := defaultConfig()
	want := expectedDefaultProviderPaths()
	paths := cfg.PathSources()
	if len(paths) != len(want) {
		t.Fatalf("paths len = %d, want %d: %#v", len(paths), len(want), paths)
	}
	for index, path := range want {
		if paths[index] != path {
			t.Fatalf("paths[%d] = %q, want %q", index, paths[index], path)
		}
	}
}

func expectedDefaultProviderPaths() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{"PATH", "/Applications", "/System/Applications", "$HOME/Applications"}
	case "linux":
		return []string{"PATH", "$HOME/.local/bin", "/usr/local/bin", "/usr/bin", "/bin", "/snap/bin"}
	case "windows":
		return []string{"PATH", "%ProgramFiles%", "%ProgramFiles(x86)%", `%LocalAppData%\Programs`}
	default:
		return []string{"PATH"}
	}
}

func TestEnsureWritesMissingConfig(t *testing.T) {
	home := t.TempDir()
	configFile := New(home)
	cfg, err := configFile.Ensure()
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if len(cfg.ProviderSources) == 0 {
		t.Fatal("Ensure() returned empty provider sources")
	}
	if cfg.Logging.Level != LogLevelInfo || !cfg.Logging.File.FileEnabled() {
		t.Fatalf("logging = %#v, want default file logging", cfg.Logging)
	}
	if _, err := os.Stat(filepath.Join(home, fileName)); err != nil {
		t.Fatalf("default config was not written: %v", err)
	}
}

func TestLoadProviderSourcesConfig(t *testing.T) {
	home := t.TempDir()
	configFile := New(home)
	configPath := filepath.Join(home, fileName)
	if err := os.WriteFile(configPath, []byte(`{
  "provider_sources": [
    {"kind": "path", "value": "PATH"},
    {"kind": "path", "value": "/Custom/Applications"}
  ],
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
	paths := cfg.PathSources()
	if len(paths) != 2 || paths[1] != "/Custom/Applications" {
		t.Fatalf("paths = %#v, want custom paths", paths)
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
	if err := os.WriteFile(configPath, []byte(`{"provider_sources":[{"kind":"path","value":"/Custom"}]}`), 0o644); err != nil {
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
	if len(loaded.ProviderSources) != len(cfg.ProviderSources) {
		t.Fatalf("loaded sources = %#v, want %#v", loaded.ProviderSources, cfg.ProviderSources)
	}
}

func TestAddProviderPathAppendsOnce(t *testing.T) {
	home := t.TempDir()
	configFile := New(home)

	cfg, added, err := configFile.AddProviderPath("/Custom")
	if err != nil {
		t.Fatalf("AddProviderPath() error = %v", err)
	}
	if !added {
		t.Fatal("AddProviderPath() added = false, want true")
	}
	cfg, added, err = configFile.AddProviderPath("/Custom")
	if err != nil {
		t.Fatalf("AddProviderPath() duplicate error = %v", err)
	}
	if added {
		t.Fatal("AddProviderPath() duplicate added = true, want false")
	}
	seen := 0
	for _, path := range cfg.PathSources() {
		if path == "/Custom" {
			seen++
		}
	}
	if seen != 1 {
		t.Fatalf("custom path count = %d, want 1 in %#v", seen, cfg.ProviderSources)
	}
}

func TestRemoveProviderPath(t *testing.T) {
	home := t.TempDir()
	configFile := New(home)
	configPath := filepath.Join(home, fileName)
	if err := os.WriteFile(configPath, []byte(`{"provider_sources":[{"kind":"path","value":"PATH"},{"kind":"path","value":"/Custom"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, removed, err := configFile.RemoveProviderPath("/Custom")
	if err != nil {
		t.Fatalf("RemoveProviderPath() error = %v", err)
	}
	if !removed {
		t.Fatal("RemoveProviderPath() removed = false, want true")
	}
	if paths := cfg.PathSources(); len(paths) != 1 || paths[0] != "PATH" {
		t.Fatalf("paths = %#v, want only PATH", paths)
	}
}

func TestRemoveProviderPathRejectsLastSource(t *testing.T) {
	home := t.TempDir()
	configFile := New(home)
	configPath := filepath.Join(home, fileName)
	if err := os.WriteFile(configPath, []byte(`{"provider_sources":[{"kind":"path","value":"PATH"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, _, err := configFile.RemoveProviderPath("PATH"); err == nil {
		t.Fatal("RemoveProviderPath() error = nil, want last source rejection")
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
  "provider_sources": [{"kind": "path", "value": "PATH"}],
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
