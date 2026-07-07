package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/spacehz-lab/cal/internal/cald/endpoint"
	"github.com/spacehz-lab/cal/internal/contract"
)

func TestNewRejectsMissingHome(t *testing.T) {
	if _, err := New(Options{}); !errors.Is(err, ErrHomeRequired) {
		t.Fatalf("New() error = %v, want ErrHomeRequired", err)
	}
}

func TestNewAppliesDefaults(t *testing.T) {
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	daemon, err := New(Options{Home: " " + t.TempDir() + " ", Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if daemon.options.Addr != defaultAddr {
		t.Fatalf("addr = %q, want %q", daemon.options.Addr, defaultAddr)
	}
	if daemon.shutdownTimeout != defaultShutdownTimeout {
		t.Fatalf("shutdown timeout = %s, want %s", daemon.shutdownTimeout, defaultShutdownTimeout)
	}
	if got := daemon.now(); !got.Equal(now) {
		t.Fatalf("now = %s, want %s", got, now)
	}
}

func TestServeWritesEndpointServesStatusAndCleansUpOnCancel(t *testing.T) {
	daemon, home := newTestDaemon(t)
	ctx, cancel := context.WithCancel(context.Background())
	errCh := serveAsync(daemon, ctx)

	record := waitForEndpoint(t, home)
	status := getStatus(t, record.BaseURL)
	if !status.Running || status.BaseURL != record.BaseURL || status.PID != os.Getpid() {
		t.Fatalf("status = %#v, endpoint = %#v", status, record)
	}
	if record.PID != os.Getpid() || record.CreatedAt == "" {
		t.Fatalf("endpoint = %#v, want pid and created_at", record)
	}

	cancel()
	if err := <-errCh; !errors.Is(err, context.Canceled) {
		t.Fatalf("Serve() error = %v, want context.Canceled", err)
	}
	assertEndpointRemoved(t, home)
}

func TestServeStopEndpointTriggersShutdown(t *testing.T) {
	daemon, home := newTestDaemon(t)
	errCh := serveAsync(daemon, context.Background())
	record := waitForEndpoint(t, home)

	response, err := http.Post(record.BaseURL+"/v1/daemon/stop", "application/json", nil)
	if err != nil {
		t.Fatalf("post stop: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("stop status = %d, want 200", response.StatusCode)
	}
	var stopped contract.DaemonStopResponse
	if err := json.NewDecoder(response.Body).Decode(&stopped); err != nil {
		t.Fatalf("decode stop response: %v", err)
	}
	if !stopped.Stopping {
		t.Fatalf("stop response = %#v, want stopping", stopped)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("Serve() error = %v, want nil", err)
	}
	assertEndpointRemoved(t, home)
}

func TestServeListenFailureLeavesNoEndpoint(t *testing.T) {
	home := t.TempDir()
	daemon, err := New(Options{Home: home, Addr: "127.0.0.1:-1"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := daemon.Serve(context.Background()); err == nil {
		t.Fatal("Serve() error = nil, want listen failure")
	}
	assertEndpointRemoved(t, home)
}

func newTestDaemon(t *testing.T) (*Daemon, string) {
	t.Helper()
	home := t.TempDir()
	daemon, err := New(Options{
		Home: home,
		Now: func() time.Time {
			return time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
		},
		ShutdownTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return daemon, home
}

func serveAsync(daemon *Daemon, ctx context.Context) <-chan error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- daemon.Serve(ctx)
	}()
	return errCh
}

func waitForEndpoint(t *testing.T, home string) endpoint.Record {
	t.Helper()
	var lastErr error
	for attempt := 0; attempt < 100; attempt++ {
		record, ok, err := endpoint.Read(home)
		if err == nil && ok {
			return record
		}
		lastErr = err
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("endpoint was not written, last error = %v", lastErr)
	return endpoint.Record{}
}

func getStatus(t *testing.T, baseURL string) contract.DaemonStatus {
	t.Helper()
	client := http.Client{Timeout: time.Second}
	response, err := client.Get(baseURL + "/v1/daemon/status")
	if err != nil {
		t.Fatalf("get status: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status code = %d, want 200", response.StatusCode)
	}
	var status contract.DaemonStatus
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	return status
}

func assertEndpointRemoved(t *testing.T, home string) {
	t.Helper()
	if _, ok, err := endpoint.Read(home); err != nil || ok {
		t.Fatalf("endpoint read = ok:%v err:%v, want missing", ok, err)
	}
}
