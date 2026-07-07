package jsonfile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAtomicWritesIndentedJSONAndCreatesDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "record.json")
	value := map[string]string{"id": "record_a"}

	if err := WriteAtomic(path, value); err != nil {
		t.Fatalf("WriteAtomic() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "{\n  \"id\": \"record_a\"\n}\n" {
		t.Fatalf("json = %q, want indented JSON", data)
	}

	var decoded map[string]string
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded["id"] != "record_a" {
		t.Fatalf("decoded id = %q, want record_a", decoded["id"])
	}
}

func TestWriteAtomicRemovesTempFileOnEncodeError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")

	if err := WriteAtomic(path, map[string]any{"bad": make(chan int)}); err == nil {
		t.Fatal("WriteAtomic() error = nil, want encode error")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("temp entries = %d, want 0", len(entries))
	}
}
