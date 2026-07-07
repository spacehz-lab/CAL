package httpserver

import (
	"net/http"
	"testing"
	"time"

	"github.com/spacehz-lab/cal/internal/contract"
)

func TestDaemonStatusReturnsCallbackValue(t *testing.T) {
	server := newTestServer(t, DaemonControl{
		Status: func() contract.DaemonStatus {
			return contract.DaemonStatus{Running: true, BaseURL: "http://127.0.0.1:1234", PID: 42}
		},
	})

	rec := serve(t, server, http.MethodGet, routeDaemonStatus, "")

	assertStatus(t, rec, http.StatusOK)
	status := decodeResponse[contract.DaemonStatus](t, rec)
	if !status.Running || status.BaseURL != "http://127.0.0.1:1234" || status.PID != 42 {
		t.Fatalf("status = %#v, want callback status", status)
	}
}

func TestDaemonStatusDefaultsToRunning(t *testing.T) {
	rec := serve(t, newTestServer(t, DaemonControl{}), http.MethodGet, routeDaemonStatus, "")

	assertStatus(t, rec, http.StatusOK)
	status := decodeResponse[contract.DaemonStatus](t, rec)
	if !status.Running {
		t.Fatalf("status = %#v, want running default", status)
	}
}

func TestDaemonStopCallsCallback(t *testing.T) {
	stopped := make(chan struct{}, 1)
	server := newTestServer(t, DaemonControl{Stop: func() { stopped <- struct{}{} }})

	rec := serve(t, server, http.MethodPost, routeDaemonStop, "")

	assertStatus(t, rec, http.StatusOK)
	response := decodeResponse[contract.DaemonStopResponse](t, rec)
	if !response.Stopping {
		t.Fatalf("response = %#v, want stopping", response)
	}
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("stop callback was not called")
	}
}

func TestDaemonStopReturnsUnavailableWhenMissing(t *testing.T) {
	rec := serve(t, newTestServer(t, DaemonControl{}), http.MethodPost, routeDaemonStop, "")

	assertStatus(t, rec, http.StatusServiceUnavailable)
	assertErrorCode(t, rec, contract.ErrorCaldUnavailable)
}
