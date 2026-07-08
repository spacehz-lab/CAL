package check

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spacehz-lab/cal/internal/model"
)

func registerArchivePredicates(c *Checker) {
	c.register(predicate{
		name:     model.VerifyPredicateArchiveContainsInput,
		subjects: []model.VerifySubjectType{model.VerifySubjectFile},
		params: []paramRule{
			{name: paramSource, required: true},
			{name: paramFormat, required: true, allowedValues: []string{formatZIP}},
		},
		run: checkArchiveContainsInput,
	})
}

func checkArchiveContainsInput(ctx *predicateContext) error {
	if format := strings.ToLower(stringParam(ctx.check.Params, paramFormat)); format != formatZIP {
		return fmt.Errorf("verify archive format %q is not supported", format)
	}
	source := pathParam(ctx.check.Params, paramSource, ctx.subject.inputs)
	sourceBytes, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}
	archive, err := zip.OpenReader(ctx.subject.path)
	if err != nil {
		return fmt.Errorf("verify archive_contains_input open zip: %w", err)
	}
	defer archive.Close()
	for _, file := range archive.File {
		if file.FileInfo().IsDir() {
			continue
		}
		content, err := readZipFile(file)
		if err != nil {
			return err
		}
		if bytes.Equal(content, sourceBytes) {
			return nil
		}
	}
	return fmt.Errorf("verify archive_contains_input failed")
}

func readZipFile(file *zip.File) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("read zip entry: %w", err)
	}
	defer reader.Close()
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read zip entry: %w", err)
	}
	return content, nil
}
