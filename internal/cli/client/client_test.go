package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spacehz-lab/cal/internal/cald/endpoint"
	"github.com/spacehz-lab/cal/internal/contract"
)

func TestNewReadsEndpointAndCallsStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != routeDaemonStatus {
			t.Fatalf("request = %s %s, want GET %s", r.Method, r.URL.Path, routeDaemonStatus)
		}
		writeJSON(t, w, contract.DaemonStatus{Running: true, BaseURL: "local", PID: 123})
	}))
	defer server.Close()

	home := t.TempDir()
	if err := endpoint.Write(home, &endpoint.Record{BaseURL: server.URL, PID: 123}); err != nil {
		t.Fatalf("endpoint.Write() error = %v", err)
	}
	client, err := New(Options{Home: home})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	status, err := client.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if !status.Running || status.PID != 123 {
		t.Fatalf("status = %#v, want running daemon", status)
	}
}

func TestNewUsesExplicitBaseURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, contract.DaemonStatus{Running: true})
	}))
	defer server.Close()

	client, err := New(Options{BaseURL: server.URL + "/"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if client.baseURL != server.URL {
		t.Fatalf("baseURL = %q, want trimmed server URL", client.baseURL)
	}
}

func TestNewDefaultsTimeoutToAtLeastTwentyMinutes(t *testing.T) {
	client, err := New(Options{BaseURL: "http://127.0.0.1:1"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if client.http.Timeout < 20*time.Minute {
		t.Fatalf("timeout = %s, want at least 20m", client.http.Timeout)
	}
}

func TestNewMissingEndpointReturnsUnavailable(t *testing.T) {
	_, err := New(Options{Home: t.TempDir()})
	assertClientError(t, err, 0, contract.ErrorCaldUnavailable)
}

func TestStatusDecodesStructuredError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(t, w, contract.ErrorResponse{Error: contract.Error{Code: contract.ErrorInvalidRequest, Message: "bad request"}})
	}))
	defer server.Close()
	client := newTestClient(t, server)

	_, err := client.Status(context.Background())
	assertClientError(t, err, http.StatusBadRequest, contract.ErrorInvalidRequest)
}

func TestStatusTransportErrorReturnsUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	baseURL := server.URL
	server.Close()
	client, err := New(Options{BaseURL: baseURL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.Status(context.Background())
	assertClientError(t, err, 0, contract.ErrorCaldUnavailable)
}

func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return client
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
}

func assertClientError(t *testing.T, err error, statusCode int, code contract.ErrorCode) {
	t.Helper()
	var clientErr *Error
	if !errors.As(err, &clientErr) {
		t.Fatalf("error = %T %[1]v, want *Error", err)
	}
	if clientErr.StatusCode != statusCode || clientErr.Code != code {
		t.Fatalf("client error = %#v, want status %d code %s", clientErr, statusCode, code)
	}
}
