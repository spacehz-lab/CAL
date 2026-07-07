package endpoint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPathReturnsEndpointPath(t *testing.T) {
	home := filepath.Join("tmp", "cal-home")
	want := filepath.Join(home, dirName, fileName)

	if got := Path(home); got != want {
		t.Fatalf("Path() = %q, want %q", got, want)
	}
}

func TestWriteReadAndRemove(t *testing.T) {
	home := t.TempDir()
	record := &Record{
		BaseURL:   "http://127.0.0.1:18080",
		PID:       1234,
		CreatedAt: "2026-07-06T12:00:00Z",
	}

	if err := Write(home, record); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	got, ok, err := Read(home)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if !ok {
		t.Fatal("Read() ok = false, want true")
	}
	if got != *record {
		t.Fatalf("Read() = %#v, want %#v", got, *record)
	}

	if err := Remove(home); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if _, err := os.Stat(Path(home)); !os.IsNotExist(err) {
		t.Fatalf("Stat() error = %v, want not exist", err)
	}
}

func TestReadMissingReturnsFalse(t *testing.T) {
	record, ok, err := Read(t.TempDir())
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if ok {
		t.Fatal("Read() ok = true, want false")
	}
	if record != (Record{}) {
		t.Fatalf("Read() record = %#v, want zero", record)
	}
}

func TestRemoveMissingSucceeds(t *testing.T) {
	if err := Remove(t.TempDir()); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
}

func TestOperationsRequireHome(t *testing.T) {
	record := &Record{BaseURL: "http://127.0.0.1:18080"}

	if _, _, err := Read(" "); err == nil {
		t.Fatal("Read() error = nil, want missing home error")
	}
	if err := Write(" ", record); err == nil {
		t.Fatal("Write() error = nil, want missing home error")
	}
	if err := Remove(" "); err == nil {
		t.Fatal("Remove() error = nil, want missing home error")
	}
}

func TestWriteValidatesRecord(t *testing.T) {
	home := t.TempDir()

	if err := Write(home, nil); err == nil {
		t.Fatal("Write(nil) error = nil, want validation error")
	}
	if err := Write(home, &Record{}); err == nil {
		t.Fatal("Write(empty) error = nil, want validation error")
	}
}

func TestReadValidatesRecord(t *testing.T) {
	home := t.TempDir()
	writeTestEndpointFile(t, home, `{"base_url":""}`)

	if _, _, err := Read(home); err == nil {
		t.Fatal("Read() error = nil, want validation error")
	}
}

func TestReadRejectsUnknownFields(t *testing.T) {
	home := t.TempDir()
	writeTestEndpointFile(t, home, `{"base_url":"http://127.0.0.1:18080","extra":true}`)

	if _, _, err := Read(home); err == nil || !strings.Contains(err.Error(), "decode endpoint") {
		t.Fatalf("Read() error = %v, want decode endpoint error", err)
	}
}

func writeTestEndpointFile(t *testing.T, home string, data string) {
	t.Helper()
	path := Path(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
