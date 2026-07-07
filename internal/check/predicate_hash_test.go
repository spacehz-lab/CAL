package check

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestHashLineMatchesPassesAndFails(t *testing.T) {
	source := writeTempFile(t, "source.txt", "hello\n")
	sum := sha256.Sum256([]byte("hello\n"))
	stdout := hex.EncodeToString(sum[:]) + "  source.txt\n"
	check := stdoutCheck(model.VerifyPredicateHashLineMatches, map[string]any{paramSource: "source", paramAlgorithm: hashSHA256Dash})

	if err := runOneCheck(check, map[string]any{"source": source}, stdout, "", 0); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if err := runOneCheck(check, map[string]any{"source": source}, "bad hash", "", 0); err == nil {
		t.Fatal("Run() error = nil, want mismatch error")
	}
}
