package check

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestFilePredicatesPassAndFail(t *testing.T) {
	jsonFile := writeTempFile(t, "report.json", `{"ok":true}`)
	emptyFile := writeTempFile(t, "empty.txt", "")
	pngFile := writeTempFileBytes(t, "image.png", []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})

	tests := []struct {
		name    string
		target  string
		check   model.VerifyCheck
		wantErr bool
	}{
		{name: "exists pass", target: jsonFile, check: fileCheck(model.VerifyPredicateExists, nil)},
		{name: "exists fail", target: filepath.Join(t.TempDir(), "missing.txt"), check: fileCheck(model.VerifyPredicateExists, nil), wantErr: true},
		{name: "non empty pass", target: jsonFile, check: fileCheck(model.VerifyPredicateNonEmpty, nil)},
		{name: "non empty fail", target: emptyFile, check: fileCheck(model.VerifyPredicateNonEmpty, nil), wantErr: true},
		{name: "format pass", target: pngFile, check: fileCheck(model.VerifyPredicateFormat, map[string]any{paramFormat: formatPNG})},
		{name: "format fail", target: jsonFile, check: fileCheck(model.VerifyPredicateFormat, map[string]any{paramFormat: formatPDF}), wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := runOneCheck(test.check, map[string]any{"target": test.target}, "", "", 0)
			if test.wantErr && err == nil {
				t.Fatal("Run() error = nil, want error")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("Run() error = %v", err)
			}
		})
	}
}

func TestFileFormatAcceptsPDFJSONAndText(t *testing.T) {
	tests := []struct {
		name   string
		format string
		path   string
	}{
		{name: "pdf", format: formatPDF, path: writeTempFile(t, "report.pdf", "%PDF-1.7\n")},
		{name: "json", format: formatJSON, path: writeTempFile(t, "report.json", `{"ok":true}`)},
		{name: "text", format: formatText, path: writeTempFile(t, "report.txt", "hello\n")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			check := fileCheck(model.VerifyPredicateFormat, map[string]any{paramFormat: test.format})
			if err := runOneCheck(check, map[string]any{"target": test.path}, "", "", 0); err != nil {
				t.Fatalf("Run() error = %v", err)
			}
		})
	}
}

func runOneCheck(check model.VerifyCheck, inputs map[string]any, stdout, stderr string, exitCode int) error {
	spec := model.VerifySpec{Level: model.VerifyLevelL2, Method: model.VerifyMethodExecute, Checks: []model.VerifyCheck{check}}
	_, err := NewChecker().Run(context.Background(), &Request{Spec: &spec, Inputs: inputs, Stdout: stdout, Stderr: stderr, ExitCode: exitCode})
	return err
}

func fileCheck(predicate model.VerifyPredicate, params map[string]any) model.VerifyCheck {
	return model.VerifyCheck{
		Subject:   model.VerifySubject{Type: model.VerifySubjectFile, Input: "target"},
		Predicate: predicate,
		Params:    params,
	}
}
