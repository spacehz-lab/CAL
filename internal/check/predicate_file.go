package check

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/spacehz-lab/cal/internal/model"
)

func registerFilePredicates(c *Checker) {
	c.register(predicate{
		name:     model.VerifyPredicateExists,
		subjects: []model.VerifySubjectType{model.VerifySubjectFile},
		run:      checkExists,
	})
	c.register(predicate{
		name:     model.VerifyPredicateNonEmpty,
		subjects: []model.VerifySubjectType{model.VerifySubjectFile, model.VerifySubjectStdout, model.VerifySubjectStderr},
		run:      checkNonEmpty,
	})
	c.register(predicate{
		name:     model.VerifyPredicateFormat,
		subjects: []model.VerifySubjectType{model.VerifySubjectFile},
		params:   []paramRule{{name: paramFormat, required: true, allowedValues: []string{formatPDF, formatPNG, formatJSON, formatText}}},
		run:      checkFormat,
	})
}

func checkExists(ctx *predicateContext) error {
	if _, err := os.Stat(ctx.subject.path); err != nil {
		return fmt.Errorf("verify %s exists failed: %w", ctx.subject.label, err)
	}
	return nil
}

func checkNonEmpty(ctx *predicateContext) error {
	if ctx.subject.path != "" {
		info, err := os.Stat(ctx.subject.path)
		if err != nil {
			return fmt.Errorf("verify %s non_empty failed: %w", ctx.subject.label, err)
		}
		if info.Size() == 0 {
			return fmt.Errorf("verify %s non_empty failed: empty artifact", ctx.subject.label)
		}
		return nil
	}
	if strings.TrimSpace(fmt.Sprint(ctx.subject.value)) == "" {
		return fmt.Errorf("verify %s non_empty failed", ctx.subject.label)
	}
	return nil
}

func checkFormat(ctx *predicateContext) error {
	format := stringParam(ctx.check.Params, paramFormat)
	content, err := os.ReadFile(ctx.subject.path)
	if err != nil {
		return fmt.Errorf("verify format read: %w", err)
	}
	switch strings.ToLower(format) {
	case formatPDF:
		if !bytes.HasPrefix(content, []byte("%PDF")) {
			return fmt.Errorf("verify format pdf failed")
		}
	case formatPNG:
		if !bytes.HasPrefix(content, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
			return fmt.Errorf("verify format png failed")
		}
	case formatJSON:
		var value any
		if err := json.Unmarshal(content, &value); err != nil {
			return fmt.Errorf("verify format json failed: %w", err)
		}
	case formatText:
		if !utf8.Valid(content) {
			return fmt.Errorf("verify format text failed")
		}
	default:
		return fmt.Errorf("verify format %q is not supported", format)
	}
	return nil
}
