package httpserver

import (
	"net/http"
	"testing"

	app "github.com/spacehz-lab/cal/internal/cald/app"
	"github.com/spacehz-lab/cal/internal/contract"
)

func TestNewRejectsMissingApp(t *testing.T) {
	if _, err := New(Options{}); err == nil {
		t.Fatal("New() error = nil, want missing app error")
	}
}

func TestServerImplementsHTTPHandler(t *testing.T) {
	application, err := app.New(app.Options{Home: t.TempDir()})
	if err != nil {
		t.Fatalf("app.New() error = %v", err)
	}
	server, err := New(Options{App: application})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	var _ http.Handler = server
}

func TestUnknownRouteReturnsJSONNotFound(t *testing.T) {
	rec := serve(t, newTestServer(t, DaemonControl{}), http.MethodGet, "/v1/missing", "")
	assertStatus(t, rec, http.StatusNotFound)
	assertErrorCode(t, rec, contract.ErrorNotFound)
}

func TestUnsupportedMethodReturnsJSONError(t *testing.T) {
	rec := serve(t, newTestServer(t, DaemonControl{}), http.MethodPut, routeProviders, "")
	assertStatus(t, rec, http.StatusMethodNotAllowed)
	assertErrorCode(t, rec, contract.ErrorInvalidRequest)
	if rec.Header().Get("Allow") == "" {
		t.Fatal("Allow header is empty, want allowed methods")
	}
}
