package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func normalizeProviderPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("provider source path is required")
	}
	return path, nil
}

func newPathSource(path string) (ProviderSource, error) {
	path, err := normalizeProviderPath(path)
	if err != nil {
		return ProviderSource{}, err
	}
	return ProviderSource{Kind: ProviderSourceKindPath, Value: path}, nil
}

func pathSources(paths []string) []ProviderSource {
	sources := make([]ProviderSource, 0, len(paths))
	for _, path := range paths {
		source, err := newPathSource(path)
		if err != nil {
			continue
		}
		sources = append(sources, source)
	}
	return sources
}

func normalizeProviderSources(sources []ProviderSource) ([]ProviderSource, error) {
	normalized := make([]ProviderSource, 0, len(sources))
	for _, source := range sources {
		switch source.Kind {
		case ProviderSourceKindPath:
			path, err := normalizeProviderPath(source.Value)
			if err != nil {
				return nil, err
			}
			normalized = append(normalized, ProviderSource{Kind: ProviderSourceKindPath, Value: path})
		default:
			return nil, fmt.Errorf("provider source kind %q is not supported", source.Kind)
		}
	}
	return normalized, nil
}

func readConfig(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config: %w", err)
	}
	defer file.Close()

	var cfg Config
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	return cfg, nil
}

func writeJSONAtomic(path string, value any) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".tmp-*.json")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)

	encoder := json.NewEncoder(temp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		temp.Close()
		return fmt.Errorf("encode config: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close config: %w", err)
	}
	if err := os.Rename(tempName, path); err != nil {
		return fmt.Errorf("rename config: %w", err)
	}
	return nil
}
