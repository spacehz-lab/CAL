package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	providersDir    = "providers"
	capabilitiesDir = "capabilities"
	discoveryDir    = "discovery"
	runsDir         = "runs"
)

// Store owns JSON persistence under one CAL home directory.
type Store struct {
	home string
}

// Open creates a store rooted at home.
func Open(home string) (*Store, error) {
	if strings.TrimSpace(home) == "" {
		return nil, fmt.Errorf("store home is required")
	}
	return &Store{home: filepath.Clean(home)}, nil
}

// Home returns the store root path.
func (s *Store) Home() string {
	return s.home
}

// Ensure creates the top-level CAL home directories.
func (s *Store) Ensure() error {
	for _, dir := range []string{providersDir, capabilitiesDir, discoveryDir, runsDir} {
		if err := os.MkdirAll(filepath.Join(s.home, dir), 0o755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}
	return nil
}
