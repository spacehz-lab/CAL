package cli

import (
	"net/http"
	"testing"

	"github.com/spacehz-lab/cal/internal/contract"
)

func TestDaemonStatusUnavailableRendersStopped(t *testing.T) {
	cmd, stdout, _ := newTestCLI(t, nil)

	if err := execute(t, cmd, "daemon", "status", "--json"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	status := decodeOutput[contract.DaemonStatus](t, stdout)
	if status.Running {
		t.Fatalf("status = %#v, want stopped", status)
	}
}

func TestDaemonStatusCallsClient(t *testing.T) {
	cmd, stdout, _ := newTestCLI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/daemon/status" {
			t.Fatalf("request = %s %s, want GET /v1/daemon/status", r.Method, r.URL.Path)
		}
		writeResponse(t, w, contract.DaemonStatus{Running: true, BaseURL: "http://127.0.0.1:1", PID: 123})
	})

	if err := execute(t, cmd, "daemon", "status", "--json"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	status := decodeOutput[contract.DaemonStatus](t, stdout)
	if !status.Running || status.PID != 123 {
		t.Fatalf("status = %#v, want running pid 123", status)
	}
}

func TestDaemonStopCallsClient(t *testing.T) {
	cmd, stdout, _ := newTestCLI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/daemon/stop" {
			t.Fatalf("request = %s %s, want POST /v1/daemon/stop", r.Method, r.URL.Path)
		}
		writeResponse(t, w, contract.DaemonStopResponse{Stopping: true})
	})

	if err := execute(t, cmd, "daemon", "stop", "--json"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	response := decodeOutput[contract.DaemonStopResponse](t, stdout)
	if !response.Stopping {
		t.Fatalf("response = %#v, want stopping", response)
	}
}
