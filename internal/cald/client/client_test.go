package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spacehz-lab/cal/internal/cald"
	"github.com/spacehz-lab/cal/internal/cald/control"
)

func TestNewReadsEndpointAndCallsStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/daemon/status" {
			t.Fatalf("request = %s %s, want GET /v1/daemon/status", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(control.Status{Running: true, Mode: "local"})
	}))
	defer server.Close()

	home := t.TempDir()
	writeEndpoint(t, home, server.URL)
	client, err := New(home)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	status, err := client.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if !status.Running || status.Mode != "local" {
		t.Fatalf("Status() = %#v, want running local", status)
	}
}

func TestNewMissingEndpointReturnsUnavailable(t *testing.T) {
	_, err := New(t.TempDir())
	var apiErr control.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "cald_unavailable" {
		t.Fatalf("New() error = %#v, want cald_unavailable", err)
	}
}

func TestClientDecodesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": control.NewAPIError("invalid_request", "bad input"),
		})
	}))
	defer server.Close()

	client := NewForEndpoint(server.URL)
	_, err := client.Status(context.Background())
	var apiErr control.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "invalid_request" || apiErr.Message != "bad input" {
		t.Fatalf("Status() error = %#v, want invalid_request", err)
	}
}

func writeEndpoint(t *testing.T, home, baseURL string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(cald.EndpointFilePath(home)), 0o700); err != nil {
		t.Fatalf("create endpoint dir: %v", err)
	}
	content, err := json.Marshal(cald.EndpointFile{BaseURL: baseURL})
	if err != nil {
		t.Fatalf("marshal endpoint: %v", err)
	}
	if err := os.WriteFile(cald.EndpointFilePath(home), content, 0o600); err != nil {
		t.Fatalf("write endpoint: %v", err)
	}
}
