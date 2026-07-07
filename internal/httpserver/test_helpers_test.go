package httpserver

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	app "github.com/spacehz-lab/cal/internal/cald/app"
	"github.com/spacehz-lab/cal/internal/contract"
)

func newTestServer(t *testing.T, daemon DaemonControl) *Server {
	t.Helper()
	application, err := app.New(app.Options{Home: t.TempDir()})
	if err != nil {
		t.Fatalf("app.New() error = %v", err)
	}
	server, err := New(Options{App: application, Daemon: daemon})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return server
}

func serve(t *testing.T, server *Server, method string, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	return rec
}

func decodeResponse[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var result T
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode response error = %v; body = %s", err, rec.Body.String())
	}
	return result
}

func assertStatus(t *testing.T, rec *httptest.ResponseRecorder, status int) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, status, rec.Body.String())
	}
}

func assertErrorCode(t *testing.T, rec *httptest.ResponseRecorder, code contract.ErrorCode) {
	t.Helper()
	response := decodeResponse[contract.ErrorResponse](t, rec)
	if response.Error.Code != code {
		t.Fatalf("error code = %s, want %s; body = %s", response.Error.Code, code, rec.Body.String())
	}
}

func writeProviderScript(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "provider.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nprintf ok\n"), 0o755); err != nil {
		t.Fatalf("write provider script: %v", err)
	}
	return path
}
