package check

import (
	"archive/zip"
	"os"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestArchiveContainsInputPassesAndFails(t *testing.T) {
	source := writeTempFile(t, "source.txt", "hello archive\n")
	target := writeTempZip(t, "target.zip", map[string]string{"source.txt": "hello archive\n"})
	bad := writeTempZip(t, "bad.zip", map[string]string{"other.txt": "other\n"})
	check := fileCheck(model.VerifyPredicateArchiveContainsInput, map[string]any{paramSource: "source", paramFormat: formatZIP})

	if err := runOneCheck(check, map[string]any{"source": source, "target": target}, "", "", 0); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if err := runOneCheck(check, map[string]any{"source": source, "target": bad}, "", "", 0); err == nil {
		t.Fatal("Run() error = nil, want mismatch error")
	}
}

func writeTempZip(t *testing.T, name string, files map[string]string) string {
	t.Helper()
	path := writeTempFile(t, name, "")
	target, err := os.Create(path)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	writer := zip.NewWriter(target)
	for filename, content := range files {
		file, err := writer.Create(filename)
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}
		if _, err := file.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	if err := target.Close(); err != nil {
		t.Fatalf("close zip file: %v", err)
	}
	return path
}
