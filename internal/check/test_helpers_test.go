package check

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempFile(t *testing.T, name string, content string) string {
	t.Helper()
	return writeTempFileBytes(t, name, []byte(content))
}

func writeTempFileBytes(t *testing.T, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}
