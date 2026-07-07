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
	tracesDir       = "traces"
	runsDir         = "runs"
	traceFileName   = "trace.json"
	jsonExt         = ".json"
)

// Store owns JSON persistence under one CAL home directory.
type Store struct {
	root string
}

// New creates a store rooted at root.
func New(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("store root is required")
	}
	return &Store{root: filepath.Clean(root)}, nil
}

// Root returns the store root path.
func (store *Store) Root() string {
	return store.root
}

// Ensure creates the top-level store directories.
func (store *Store) Ensure() error {
	for _, dir := range []string{providersDir, capabilitiesDir, tracesDir, runsDir} {
		if err := os.MkdirAll(filepath.Join(store.root, dir), 0o755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}
	return nil
}
