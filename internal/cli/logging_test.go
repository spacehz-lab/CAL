package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/logging"
)

func TestDiscoverScanInfoLogsUseStderr(t *testing.T) {
	t.Setenv(logging.EnvLogLevel, "")
	home := t.TempDir()
	startCLITestCald(t, home)
	installCLITestVerifier(t, home, "file_parse_pdf", pdfMagicVerifierScript())
	if err := os.WriteFile(filepath.Join(home, "config.json"), []byte(`{"logging":{"level":"info","file":{"enabled":false}}}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	providerPath := writeAcquisitionScript(t)
	store := newTestStoreWithHome(t, home)
	if err := store.PutProvider(testCLIProvider("provider_cli", providerPath)); err != nil {
		t.Fatalf("PutProvider() error = %v", err)
	}
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd := NewRootCommand(Config{Home: home, Out: &out, Err: &stderr})
	cmd.SetArgs([]string{"discovery", "run", "--provider-id", "provider_cli", "--mode", "rules", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("discovery run error = %v\nstderr=%s\nstdout=%s", err, stderr.String(), out.String())
	}

	var result map[string]any
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("stdout is not clean JSON: %v\nstdout=%s\nstderr=%s", err, out.String(), stderr.String())
	}
	if strings.Contains(out.String(), "discovery acquisition completed") || strings.Contains(out.String(), "level=") {
		t.Fatalf("stdout contains logs: %q", out.String())
	}
}
