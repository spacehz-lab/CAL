package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spacehz-lab/cal/pkg/jsonfile"
)

// FileName is the local CAL config file name.
const FileName = "config.json"

// File reads and writes config.json under one CAL home.
type File struct {
	path string
}

// Config is the user-editable local CAL configuration.
type Config struct {
	LLM     *LLMConfig    `json:"llm,omitempty"`
	Logging LoggingConfig `json:"logging,omitempty"`
}

// NewFile creates a config file handle rooted at one CAL home.
func NewFile(home string) *File {
	return &File{path: filepath.Join(home, FileName)}
}

// Path returns the concrete config file path.
func (file *File) Path() string {
	return file.path
}

// Load reads, defaults, and validates config.json.
func (file *File) Load() (*Config, error) {
	cfg, err := readConfig(file.path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return nil, err
	}
	cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Save validates and writes config.json.
func (file *File) Save(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is required")
	}
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return err
	}
	return jsonfile.WriteAtomic(file.path, cfg)
}

// Ensure writes the default config file when it is missing.
func (file *File) Ensure() (*Config, error) {
	if _, err := os.Stat(file.path); err == nil {
		return file.Load()
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("stat config: %w", err)
	}
	cfg := Default()
	if err := file.Save(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Default returns a fresh default configuration.
func Default() *Config {
	return (&Config{}).WithDefaults()
}

// WithDefaults applies product defaults in place and returns cfg.
func (cfg *Config) WithDefaults() *Config {
	if cfg == nil {
		return Default()
	}
	if cfg.LLM != nil {
		cfg.LLM.withDefaults()
	}
	cfg.Logging = cfg.Logging.withDefaults()
	return cfg
}

// Validate checks the config contract.
func (cfg *Config) Validate() error {
	if cfg == nil {
		return fmt.Errorf("config is required")
	}
	if cfg.LLM != nil {
		if err := cfg.LLM.Validate(); err != nil {
			return err
		}
	}
	if err := cfg.Logging.Validate(); err != nil {
		return err
	}
	return nil
}

func readConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer file.Close()

	var cfg Config
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	return &cfg, nil
}
