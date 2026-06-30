package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const fileName = "config.json"

// File reads and writes config.json for one CAL home.
type File struct {
	path string
}

// Config is the user-editable local CAL configuration.
type Config struct {
	LLM     *LLMSettings `json:"llm,omitempty"`
	Logging Logging      `json:"logging,omitempty"`
}

func defaultConfig() Config {
	return Config{
		Logging: DefaultLogging(),
	}
}

// New creates a config file rooted at one CAL home.
func New(calHome string) *File {
	return &File{path: filepath.Join(calHome, fileName)}
}

// Load reads CAL configuration.
func (f *File) Load() (Config, error) {
	cfg, err := readConfig(f.path)
	if errors.Is(err, os.ErrNotExist) {
		return defaultConfig(), nil
	}
	if err != nil {
		return Config{}, err
	}
	if err := cfg.applyDefaults(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Ensure writes default configuration when config.json is missing.
func (f *File) Ensure() (Config, error) {
	if _, err := os.Stat(f.path); err == nil {
		return f.Load()
	} else if !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("stat config: %w", err)
	}
	return f.reset()
}

func (f *File) reset() (Config, error) {
	cfg := defaultConfig()
	if err := f.save(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (f *File) save(cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	return writeJSONAtomic(f.path, cfg)
}
