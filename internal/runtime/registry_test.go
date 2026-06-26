package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

func TestRegistryStartsEmptyWithoutCALHome(t *testing.T) {
	useRuntimeHome(t)

	registry := NewRegistry()
	if registry.Supports("file_exists") {
		t.Fatal("Supports(file_exists) = true, want no packaged default verifier")
	}
}

func TestRegistryLoadsScriptVerifier(t *testing.T) {
	useRuntimeHome(t)
	dir := filepath.Join(t.TempDir(), "verifiers")
	writeScriptVerifier(t, dir, "always_passes", `import json
import sys

request = json.load(sys.stdin)
verifier = request["verifier"]
print(json.dumps({
    "passed": True,
    "evidence": [{
        "id": "evidence_custom",
        "type": verifier["id"],
        "content": {"source": "script"},
    }],
    "outputs": request["inputs"],
}))
`)

	registry := NewRegistry()
	if err := registry.LoadScriptVerifiers(dir); err != nil {
		t.Fatalf("LoadScriptVerifiers() error = %v", err)
	}
	if !registry.Supports("always_passes") {
		t.Fatal("Supports(always_passes) = false, want true")
	}
	evidence, outputs, err := registry.Verify(context.Background(), core.Verifier{ID: "always_passes"}, map[string]any{"target": "ok"})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if len(evidence) != 1 || evidence[0].Type != "always_passes" || outputs["target"] != "ok" {
		t.Fatalf("Verify() = %#v, %#v, want custom evidence and outputs", evidence, outputs)
	}
}

func TestRegistryRejectsScriptVerifierTimeout(t *testing.T) {
	useRuntimeHome(t)
	dir := filepath.Join(t.TempDir(), "verifiers")
	writeScriptVerifier(t, dir, "too_slow", `import time
time.sleep(2)
`)
	metaPath := filepath.Join(dir, "too_slow", "meta.json")
	if err := os.WriteFile(metaPath, []byte(`{"id":"too_slow","runtime":"python3","entry":"verify.py","timeout_ms":50}`), 0o644); err != nil {
		t.Fatalf("write timeout metadata: %v", err)
	}

	registry := NewRegistry()
	if err := registry.LoadScriptVerifiers(dir); err != nil {
		t.Fatalf("LoadScriptVerifiers() error = %v", err)
	}
	_, _, err := registry.Verify(context.Background(), core.Verifier{ID: "too_slow"}, map[string]any{})
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("Verify() error = %v, want timeout", err)
	}
}

func TestRegistryRejectsInvalidVerifierID(t *testing.T) {
	useRuntimeHome(t)
	dir := filepath.Join(t.TempDir(), "verifiers")
	writeScriptVerifier(t, dir, "bad.id", "print('{}')\n")

	registry := NewRegistry()
	err := registry.LoadScriptVerifiers(dir)
	if err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("LoadScriptVerifiers() error = %v, want invalid verifier id error", err)
	}
}

func TestRegistryVerifyRejectsUnknownVerifier(t *testing.T) {
	_, _, err := Registry{verifiers: map[string]scriptVerifier{}}.Verify(context.Background(), core.Verifier{ID: "missing"}, nil)
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("Verify() error = %v, want unsupported verifier error", err)
	}
}

func TestRegistryHelpersNormalizeAndJoinErrors(t *testing.T) {
	if got := stderrSuffix(" \nproblem\n "); got != ": problem" {
		t.Fatalf("stderrSuffix() = %q, want suffix", got)
	}
	if got := stderrSuffix(" \n "); got != "" {
		t.Fatalf("stderrSuffix(blank) = %q, want empty", got)
	}
	if err := joinVerifierError(nil, errors.New("second")); err == nil || err.Error() != "second" {
		t.Fatalf("joinVerifierError(nil, second) = %v, want second", err)
	}
	joined := joinVerifierError(errors.New("first"), errors.New("second"))
	if joined == nil || !strings.Contains(joined.Error(), "first") || !strings.Contains(joined.Error(), "second") {
		t.Fatalf("joinVerifierError() = %v, want both errors", joined)
	}

	values := normalizeJSONMap(map[string]any{
		"int":   json.Number("7"),
		"float": json.Number("1.5"),
		"bad":   json.Number("not_number"),
		"nested": map[string]any{
			"value": json.Number("2"),
		},
		"list": []any{json.Number("3")},
	})
	if values["int"] != 7 || values["float"] != 1.5 || values["bad"] != "not_number" {
		t.Fatalf("normalized numbers = %#v, want int, float, and string fallback", values)
	}
	if nested := values["nested"].(map[string]any); nested["value"] != 2 {
		t.Fatalf("nested = %#v, want normalized value", nested)
	}
	if list := values["list"].([]any); list[0] != 3 {
		t.Fatalf("list = %#v, want normalized item", list)
	}
	if verifierRuntime("custom-runtime") != "custom-runtime" {
		t.Fatal("verifierRuntime(custom-runtime) did not return custom runtime")
	}
}

func TestVerifierRuntimeUsesPathOnly(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	if got := verifierRuntime("python3"); got != "python3" {
		t.Fatalf("verifierRuntime(python3) = %q, want unresolved runtime name", got)
	}
}

func useRuntimeHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("CAL_HOME", home)
	return home
}

func writeScriptVerifier(t *testing.T, root, verifierType, script string) {
	t.Helper()
	dir := filepath.Join(root, verifierType)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create verifier dir: %v", err)
	}
	meta := `{"id":` + strconv.Quote(verifierType) + `,"runtime":"python3","entry":"verify.py","timeout_ms":3000}`
	if err := os.WriteFile(filepath.Join(dir, "meta.json"), []byte(meta), 0o644); err != nil {
		t.Fatalf("write verifier metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "verify.py"), []byte(script), 0o644); err != nil {
		t.Fatalf("write verifier script: %v", err)
	}
}
