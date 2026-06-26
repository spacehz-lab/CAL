package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
)

func TestInstallVerifierInstallsExecutableVerifier(t *testing.T) {
	home := useRuntimeHome(t)
	target := filepath.Join(t.TempDir(), "target.txt")
	if err := os.WriteFile(target, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	err := InstallVerifier(GeneratedVerifierPackage{
		ID:          "verifier_contains_ok",
		Description: "Passes when the target file contains ok.",
		VerifyPY: `import json
import sys

request = json.load(sys.stdin)
target = request["inputs"]["target"]
with open(target, "r", encoding="utf-8") as handle:
    content = handle.read()
print(json.dumps({
    "passed": "ok" in content,
    "evidence": [{"id": "verifier_contains_ok", "type": "verifier_contains_ok"}],
    "outputs": {"target": target},
}))
`,
	})
	if err != nil {
		t.Fatalf("InstallVerifier() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, verifierDirName, "verifier_contains_ok", "meta.json")); err != nil {
		t.Fatalf("generated metadata missing: %v", err)
	}

	registry := NewRegistry()
	evidence, outputs, err := registry.Verify(context.Background(), core.Verifier{ID: "verifier_contains_ok"}, map[string]any{"target": target})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if len(evidence) != 1 || evidence[0].ID != "verifier_contains_ok" || outputs["target"] != target {
		t.Fatalf("Verify() = %#v, %#v, want generated evidence", evidence, outputs)
	}
}

func TestInstallVerifierRejectsInvalidInput(t *testing.T) {
	useRuntimeHome(t)
	for _, test := range []struct {
		name string
		pkg  GeneratedVerifierPackage
	}{
		{name: "invalid id", pkg: GeneratedVerifierPackage{ID: "bad.id", VerifyPY: "print('{}')\n"}},
		{name: "empty script", pkg: GeneratedVerifierPackage{ID: "empty_script", VerifyPY: " \n\t"}},
		{name: "oversized script", pkg: GeneratedVerifierPackage{ID: "large_script", VerifyPY: strings.Repeat("x", 32*1024+1)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := InstallVerifier(test.pkg); err == nil {
				t.Fatal("InstallVerifier() error = nil, want rejection")
			}
		})
	}
}

func TestInstallVerifierRejectsUnavailableHomeAndExistingVerifier(t *testing.T) {
	missingHome := filepath.Join(t.TempDir(), "missing")
	t.Setenv("CAL_HOME", missingHome)
	err := InstallVerifier(GeneratedVerifierPackage{ID: "verifier_missing_home", VerifyPY: "print('{}')\n"})
	if err == nil || !strings.Contains(err.Error(), "not available") {
		t.Fatalf("InstallVerifier() error = %v, want unavailable home error", err)
	}

	home := useRuntimeHome(t)
	writeScriptVerifier(t, filepath.Join(home, verifierDirName), "local_existing", "print('{}')\n")
	err = InstallVerifier(GeneratedVerifierPackage{ID: "local_existing", VerifyPY: "print('{}')\n"})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("InstallVerifier(local) error = %v, want existing verifier error", err)
	}
}
