package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRotatingWriterRotatesAndDeletesOldFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "calctl.log")
	writer, err := newRotatingWriter(path, 10, 2)
	if err != nil {
		t.Fatalf("newRotatingWriter() error = %v", err)
	}

	for _, line := range []string{"first line\n", "second line\n", "third line\n", "fourth line\n"} {
		if _, err := writer.Write([]byte(line)); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
	}

	assertFileContains(t, path, "fourth line")
	assertFileContains(t, rotatedPath(path, 1), "third line")
	assertFileContains(t, rotatedPath(path, 2), "second line")
	if _, err := os.Stat(rotatedPath(path, 3)); !os.IsNotExist(err) {
		t.Fatalf("old rotated file exists err=%v, want removed", err)
	}
}

func TestRotatingWriterRejectsInvalidLimits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "calctl.log")
	if _, err := newRotatingWriter(path, 0, 1); err == nil {
		t.Fatal("newRotatingWriter() error = nil, want max bytes error")
	}
	if _, err := newRotatingWriter(path, 1, 0); err == nil {
		t.Fatal("newRotatingWriter() error = nil, want max files error")
	}
}

func assertFileContains(t *testing.T, path string, want string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(content), want) {
		t.Fatalf("%s = %q, want %q", path, content, want)
	}
}
