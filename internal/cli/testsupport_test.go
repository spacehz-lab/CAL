package cli

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spacehz-lab/cal/internal/cald"
	"github.com/spacehz-lab/cal/internal/cald/control"
	"github.com/spacehz-lab/cal/internal/cald/httpapi"
	calstore "github.com/spacehz-lab/cal/internal/store"
)

func startCLITestCald(t *testing.T, home string) {
	t.Helper()
	svc, err := control.NewService(home)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	server := httptest.NewServer(httpapi.NewRouter(httpapi.RouterConfig{
		Service: svc,
		Status: control.Status{
			Running:  true,
			Mode:     "local",
			Endpoint: serverPlaceholder,
		},
	}))
	t.Cleanup(server.Close)
	writeCLITestEndpoint(t, home, server.URL)
}

const serverPlaceholder = "http://127.0.0.1:0"

func writeCLITestEndpoint(t *testing.T, home, baseURL string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(cald.EndpointFilePath(home)), 0o700); err != nil {
		t.Fatalf("create endpoint dir: %v", err)
	}
	content, err := json.Marshal(cald.EndpointFile{BaseURL: baseURL})
	if err != nil {
		t.Fatalf("encode endpoint: %v", err)
	}
	if err := os.WriteFile(cald.EndpointFilePath(home), content, 0o600); err != nil {
		t.Fatalf("write endpoint: %v", err)
	}
}

func newTestStoreWithHome(t *testing.T, home string) *calstore.Store {
	t.Helper()
	store, err := calstore.Open(home)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := store.Ensure(); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	return store
}
