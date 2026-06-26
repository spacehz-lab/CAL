package cli

import (
	"strings"
	"testing"

	"github.com/spacehz-lab/cal/internal/core"
	calstore "github.com/spacehz-lab/cal/internal/store"
)

func TestEvalCommandTextAndJSON(t *testing.T) {
	home := t.TempDir()
	startCLITestCald(t, home)
	store, err := calstore.Open(home)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := store.PutProvider(core.Provider{ID: "provider_cli", Kind: core.ProviderKindCLI, Path: "/tmp/fake"}); err != nil {
		t.Fatalf("PutProvider() error = %v", err)
	}

	output, err := executeRoot(home, "eval")
	if err != nil {
		t.Fatalf("eval text error = %v\n%s", err, output)
	}
	if !strings.Contains(output, "providers=1") {
		t.Fatalf("eval text output = %q, want provider count", output)
	}

	output, err = executeRoot(home, "eval", "--json")
	if err != nil {
		t.Fatalf("eval json error = %v\n%s", err, output)
	}
	if !strings.Contains(output, `"providers": 1`) {
		t.Fatalf("eval json output = %q, want provider count", output)
	}
}
