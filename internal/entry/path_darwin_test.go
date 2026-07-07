//go:build darwin

package entry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spacehz-lab/cal/internal/model"
)

func TestResolveProviderPathAcceptsAppBundle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Preview.app")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("mkdir app bundle: %v", err)
	}

	provider, err := resolveProviderPath(path)
	if err != nil {
		t.Fatalf("resolveProviderPath() error = %v", err)
	}
	if provider.Kind != model.ProviderKindApp || provider.Name != "Preview" {
		t.Fatalf("provider = %#v, want app bundle provider", provider)
	}
}
