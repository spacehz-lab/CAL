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

// ProviderSourceKind identifies a configured provider source type.
type ProviderSourceKind string

const (
	// ProviderSourceKindPath identifies a local path source.
	ProviderSourceKindPath ProviderSourceKind = "path"
)

// ProviderSource configures one place where provider entries can be found.
type ProviderSource struct {
	Kind  ProviderSourceKind `json:"kind"`
	Value string             `json:"value"`
}

// Config is the user-editable local CAL configuration.
type Config struct {
	ProviderSources []ProviderSource `json:"provider_sources"`
	LLM             *LLMSettings     `json:"llm,omitempty"`
	Logging         Logging          `json:"logging,omitempty"`
}

func defaultConfig() Config {
	return Config{
		ProviderSources: pathSources(defaultProviderPaths()),
		Logging:         DefaultLogging(),
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
	sources, err := normalizeProviderSources(cfg.ProviderSources)
	if err != nil {
		return Config{}, err
	}
	cfg.ProviderSources = sources
	if len(cfg.ProviderSources) == 0 {
		return Config{}, fmt.Errorf("provider sources are required")
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
	sources, err := normalizeProviderSources(cfg.ProviderSources)
	if err != nil {
		return err
	}
	cfg.ProviderSources = sources
	if len(cfg.ProviderSources) == 0 {
		return fmt.Errorf("provider sources are required")
	}
	if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	return writeJSONAtomic(f.path, cfg)
}

// PathSources returns configured path source values in order.
func (cfg Config) PathSources() []string {
	paths := make([]string, 0, len(cfg.ProviderSources))
	for _, source := range cfg.ProviderSources {
		if source.Kind == ProviderSourceKindPath {
			paths = append(paths, source.Value)
		}
	}
	return paths
}

// AddProviderPath adds one configured path provider source if it is absent.
func (f *File) AddProviderPath(path string) (Config, bool, error) {
	source, err := newPathSource(path)
	if err != nil {
		return Config{}, false, err
	}

	cfg, err := f.Ensure()
	if err != nil {
		return Config{}, false, err
	}
	for _, existing := range cfg.ProviderSources {
		if existing == source {
			return cfg, false, nil
		}
	}
	cfg.ProviderSources = append(cfg.ProviderSources, source)
	if err := f.save(cfg); err != nil {
		return Config{}, false, err
	}
	return cfg, true, nil
}

// RemoveProviderPath removes one configured path provider source.
func (f *File) RemoveProviderPath(path string) (Config, bool, error) {
	source, err := newPathSource(path)
	if err != nil {
		return Config{}, false, err
	}

	cfg, err := f.Ensure()
	if err != nil {
		return Config{}, false, err
	}
	sources := make([]ProviderSource, 0, len(cfg.ProviderSources))
	removed := false
	for _, existing := range cfg.ProviderSources {
		if existing == source {
			removed = true
			continue
		}
		sources = append(sources, existing)
	}
	if !removed {
		return cfg, false, nil
	}
	if len(sources) == 0 {
		return Config{}, false, fmt.Errorf("cannot remove the last provider source")
	}
	cfg.ProviderSources = sources
	if err := f.save(cfg); err != nil {
		return Config{}, false, err
	}
	return cfg, true, nil
}
