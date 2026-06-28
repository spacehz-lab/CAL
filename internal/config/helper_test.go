package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadConfigRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), fileName)
	if err := os.WriteFile(path, []byte(`{"unexpected": true}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := readConfig(path); err == nil || !strings.Contains(err.Error(), "decode config") {
		t.Fatalf("readConfig() error = %v, want decode config error", err)
	}
}

func TestReadConfigPreservesOpenNotExistError(t *testing.T) {
	_, err := readConfig(filepath.Join(t.TempDir(), fileName))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("readConfig() error = %v, want os.ErrNotExist", err)
	}
}

func TestWriteJSONAtomicWritesReadableConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), fileName)
	want := Config{Logging: DefaultLogging()}
	if err := writeJSONAtomic(path, want); err != nil {
		t.Fatalf("writeJSONAtomic() error = %v", err)
	}

	got, err := readConfig(path)
	if err != nil {
		t.Fatalf("readConfig() error = %v", err)
	}
	if got.Logging.Level != want.Logging.Level {
		t.Fatalf("config = %#v, want %#v", got, want)
	}
}
