package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeProviderPathTrimsWhitespace(t *testing.T) {
	path, err := normalizeProviderPath("  /Applications  ")
	if err != nil {
		t.Fatalf("normalizeProviderPath() error = %v", err)
	}
	if path != "/Applications" {
		t.Fatalf("path = %q, want trimmed path", path)
	}
}

func TestNormalizeProviderPathRejectsBlank(t *testing.T) {
	if _, err := normalizeProviderPath(" \t\n "); err == nil {
		t.Fatal("normalizeProviderPath() error = nil, want blank path rejection")
	}
}

func TestNewPathSource(t *testing.T) {
	source, err := newPathSource("  PATH  ")
	if err != nil {
		t.Fatalf("newPathSource() error = %v", err)
	}
	if source.Kind != ProviderSourceKindPath || source.Value != "PATH" {
		t.Fatalf("source = %#v, want path source", source)
	}
}

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
	want := Config{ProviderSources: []ProviderSource{
		{Kind: ProviderSourceKindPath, Value: "PATH"},
		{Kind: ProviderSourceKindPath, Value: "/Applications"},
	}}
	if err := writeJSONAtomic(path, want); err != nil {
		t.Fatalf("writeJSONAtomic() error = %v", err)
	}

	got, err := readConfig(path)
	if err != nil {
		t.Fatalf("readConfig() error = %v", err)
	}
	if len(got.ProviderSources) != len(want.ProviderSources) {
		t.Fatalf("sources len = %d, want %d", len(got.ProviderSources), len(want.ProviderSources))
	}
	for index, source := range want.ProviderSources {
		if got.ProviderSources[index] != source {
			t.Fatalf("sources[%d] = %#v, want %#v", index, got.ProviderSources[index], source)
		}
	}
}
