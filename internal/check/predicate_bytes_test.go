package check

import (
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestBytesEqualTransformPassesAndFails(t *testing.T) {
	source := writeTempFile(t, "source.txt", "hello\n")
	encoded := writeTempFile(t, "encoded.txt", "aGVsbG8K")
	bad := writeTempFile(t, "bad.txt", "bad")

	check := fileCheck(model.VerifyPredicateBytesEqualTransform, map[string]any{paramSource: "source", paramTransform: transformBase64Encode})
	if err := runOneCheck(check, map[string]any{"source": source, "target": encoded}, "", "", 0); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if err := runOneCheck(check, map[string]any{"source": source, "target": bad}, "", "", 0); err == nil {
		t.Fatal("Run() error = nil, want mismatch error")
	}
}

func TestBytesEqualTransformDecodePasses(t *testing.T) {
	source := writeTempFile(t, "encoded.txt", "aGVsbG8K")
	decoded := writeTempFile(t, "decoded.txt", "hello\n")
	check := fileCheck(model.VerifyPredicateBytesEqualTransform, map[string]any{paramSource: "source", paramTransform: transformBase64Decode})
	if err := runOneCheck(check, map[string]any{"source": source, "target": decoded}, "", "", 0); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}
