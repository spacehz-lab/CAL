package check

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/spacehz-lab/cal/internal/model"
)

func registerHashPredicates(c *Checker) {
	c.register(predicate{
		name:     model.VerifyPredicateHashLineMatches,
		subjects: []model.VerifySubjectType{model.VerifySubjectFile, model.VerifySubjectStdout, model.VerifySubjectStderr},
		params: []paramRule{
			{name: paramSource, required: true},
			{name: paramAlgorithm, required: true, allowedValues: []string{hashSHA1, hashSHA256, hashSHA1Dash, hashSHA256Dash, hashSHA1Under, hashSHA256Under, hashSHA1Space, hashSHA256Space}},
		},
		run: checkHashLineMatches,
	})
}

func checkHashLineMatches(ctx *predicateContext) error {
	source := pathParam(ctx.check.Params, paramSource, ctx.subject.inputs)
	algorithm := stringParam(ctx.check.Params, paramAlgorithm)
	sourceBytes, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}
	want, err := hashBytes(sourceBytes, algorithm)
	if err != nil {
		return err
	}
	text, err := subjectText(ctx.subject)
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
	case hashSHA1:
		sum := sha1.Sum(content)
		return hex.EncodeToString(sum[:]), nil
	case hashSHA256:
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
