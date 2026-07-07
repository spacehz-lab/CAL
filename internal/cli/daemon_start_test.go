package cli

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spacehz-lab/cal/internal/cald/endpoint"
	"github.com/spacehz-lab/cal/internal/contract"
)

func TestDaemonStartCommandRendersRunningStatus(t *testing.T) {
	cmd, stdout, _ := newTestCLI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/daemon/status" {
			t.Fatalf("request = %s %s, want GET /v1/daemon/status", r.Method, r.URL.Path)
		}
		writeResponse(t, w, contract.DaemonStatus{Running: true, BaseURL: "http://127.0.0.1:1", PID: 123})
	})

	if err := execute(t, cmd, "daemon", "start", "--json"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	status := decodeOutput[contract.DaemonStatus](t, stdout)
	if !status.Running || status.PID != 123 {
		t.Fatalf("status = %#v, want running pid 123", status)
	}
}

func TestDaemonStarterReusesRunningDaemon(t *testing.T) {
	home := t.TempDir()
	server := newStatusServer(t, contract.DaemonStatus{Running: true, BaseURL: "local", PID: 456})
	if err := endpoint.Write(home, &endpoint.Record{BaseURL: server.URL, PID: 456}); err != nil {
		t.Fatalf("endpoint.Write() error = %v", err)
	}
	starter := &daemonStarter{
		home:    home,
		env:     []string{},
		path:    filepath.Join(t.TempDir(), "missing-cald"),
		logPath: filepath.Join(t.TempDir(), "cald.log"),
		timeout: time.Millisecond,
	}

	status, err := starter.Start(context.Background())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if !status.Running || status.PID != 456 {
		t.Fatalf("status = %#v, want running pid 456", status)
	}
}

func TestDaemonStarterReportsStartFailure(t *testing.T) {
	starter := &daemonStarter{
		home:    t.TempDir(),
		env:     []string{},
		path:    filepath.Join(t.TempDir(), "missing-cald"),
		logPath: filepath.Join(t.TempDir(), "cald.log"),
		timeout: time.Millisecond,
	}

	_, err := starter.Start(context.Background())
	if err == nil || !strings.Contains(err.Error(), "start cald") {
		t.Fatalf("Start() error = %v, want start cald failure", err)
	}
}

func TestEnvWithHomeOverridesExistingValue(t *testing.T) {
	env := envWithHome([]string{"A=1", envHome + "=/old"}, "/new")

	if got := envValue(env, envHome); got != "/new" {
		t.Fatalf("%s = %q, want /new", envHome, got)
	}
}

func newStatusServer(t *testing.T, status contract.DaemonStatus) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/daemon/status" {
			t.Fatalf("request = %s %s, want GET /v1/daemon/status", r.Method, r.URL.Path)
		}
		writeResponse(t, w, status)
	}))
	t.Cleanup(server.Close)
	return server
}
