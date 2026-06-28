package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

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
