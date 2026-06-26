package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spacehz-lab/cal/internal/cald/control"
)

func TestDaemonStarterReusesRunningEndpoint(t *testing.T) {
	home := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if err := json.NewEncoder(w).Encode(control.Status{
			Running:  true,
			Mode:     "local",
			PID:      123,
			Endpoint: "http://127.0.0.1:1234",
		}); err != nil {
			t.Fatalf("encode status: %v", err)
		}
	}))
	defer server.Close()
	writeCLITestEndpoint(t, home, server.URL)

	status, err := (daemonStarter{
		cfg:      Config{Home: home},
		caldPath: "/missing/cald",
	}).Start(context.Background())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if !status.Running || status.PID != 123 {
		t.Fatalf("Start() status = %#v, want existing running endpoint", status)
	}
}

func TestDaemonStarterMissingExecutableReturnsCodedError(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := (daemonStarter{cfg: Config{Home: t.TempDir()}}).Start(context.Background())
	commandErr, ok := commandErrorFrom(err)
	if !ok || commandErr.Code != string(commandErrorCaldStartFailed) {
		t.Fatalf("Start() error = %#v, want cald_start_failed", err)
	}
}
