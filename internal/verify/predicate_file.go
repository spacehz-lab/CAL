package verify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/spacehz-lab/cal/internal/core"
)

func checkExists(_ core.VerifyCheck, subject checkSubject) error {
	if _, err := os.Stat(subject.path); err != nil {
		return fmt.Errorf("verify %s exists failed: %w", subject.label, err)
	}
	return nil
}

func checkNonEmpty(_ core.VerifyCheck, subject checkSubject) error {
	if subject.path != "" {
		info, err := os.Stat(subject.path)
		if err != nil {
			return fmt.Errorf("verify %s non_empty failed: %w", subject.label, err)
		}
		if info.Size() == 0 {
			return fmt.Errorf("verify %s non_empty failed: empty artifact", subject.label)
		}
		return nil
	}
	if strings.TrimSpace(fmt.Sprint(subject.value)) == "" {
		return fmt.Errorf("verify %s non_empty failed", subject.label)
	}
	return nil
}

func checkFormatPredicate(check core.VerifyCheck, subject checkSubject) error {
	format := stringParam(check.Params, "format")
	if format == "" {
		return fmt.Errorf("verify format requires params.format")
	}
	content, err := os.ReadFile(subject.path)
	if err != nil {
		return fmt.Errorf("verify format read: %w", err)
	}
	switch strings.ToLower(format) {
	case "pdf":
		if !bytes.HasPrefix(content, []byte("%PDF")) {
			return fmt.Errorf("verify format pdf failed")
		}
	case "png":
		if !bytes.HasPrefix(content, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
			return fmt.Errorf("verify format png failed")
		}
	case "json":
		var value any
		if err := json.Unmarshal(content, &value); err != nil {
			return fmt.Errorf("verify format json failed: %w", err)
		}
	case "text":
		if !utf8.Valid(content) {
			return fmt.Errorf("verify format text failed")
		}
	default:
		return fmt.Errorf("verify format %q is not supported", format)
	}
	return nil
}
