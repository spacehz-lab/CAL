package verify

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/spacehz-lab/cal/internal/core"
)

func checkHashLineMatches(check core.VerifyCheck, subject checkSubject) error {
	source := pathParam(check.Params, "source", subject.inputs)
	algorithm := stringParam(check.Params, "algorithm")
	if source == "" || algorithm == "" {
		return fmt.Errorf("verify hash_line_matches requires source and algorithm")
	}
	sourceBytes, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}
	want, err := hashBytes(sourceBytes, algorithm)
	if err != nil {
		return err
	}
	text, err := subjectText(subject)
	if err != nil {
		return err
	}
	if !strings.Contains(strings.ToLower(text), want) {
		return fmt.Errorf("verify hash_line_matches failed")
	}
	return nil
}

func hashBytes(content []byte, algorithm string) (string, error) {
	switch normalizeHashAlgorithm(algorithm) {
	case "sha1":
		sum := sha1.Sum(content)
		return hex.EncodeToString(sum[:]), nil
	case "sha256":
		sum := sha256.Sum256(content)
		return hex.EncodeToString(sum[:]), nil
	default:
		return "", fmt.Errorf("verify hash algorithm %q is not supported", algorithm)
	}
}

func normalizeHashAlgorithm(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "-", "")
	normalized = strings.ReplaceAll(normalized, "_", "")
	normalized = strings.ReplaceAll(normalized, " ", "")
	return normalized
}
