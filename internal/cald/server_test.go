package cald

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestWriteConnectionFilesWritesEndpoint(t *testing.T) {
	home := t.TempDir()
	status := Status{
		PID:       123,
		Endpoint:  "http://127.0.0.1:12345",
		StartedAt: "2026-06-24T10:00:00Z",
	}

	if err := writeConnectionFiles(home, status); err != nil {
		t.Fatalf("writeConnectionFiles() error = %v", err)
	}
	endpoint := readEndpointFile(t, home)
	if endpoint.PID != status.PID || endpoint.BaseURL != status.Endpoint || endpoint.StartedAt != status.StartedAt {
		t.Fatalf("endpoint = %#v, want status-derived endpoint", endpoint)
	}
}

func TestServeWritesEndpointAndServesStatus(t *testing.T) {
	home := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- serve(ctx, home)
	}()

	endpoint := waitForEndpoint(t, home)
	statusURL := strings.TrimRight(endpoint.BaseURL, "/") + "/v1/daemon/status"
	client := http.Client{Timeout: time.Second}
	var lastErr error
	for attempt := 0; attempt < 50; attempt++ {
		resp, err := client.Get(statusURL)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				cancel()
				if err := <-errCh; err != nil && err != context.Canceled {
					t.Fatalf("serve() error = %v, want nil or context.Canceled", err)
				}
				if _, err := os.Stat(EndpointFilePath(home)); !os.IsNotExist(err) {
					t.Fatalf("endpoint file exists after shutdown: %v", err)
				}
				return
			}
			lastErr = nil
		} else {
			lastErr = err
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	_ = <-errCh
	t.Fatalf("cald status endpoint did not become ready, last err = %v", lastErr)
}

func waitForEndpoint(t *testing.T, home string) EndpointFile {
	t.Helper()
	for attempt := 0; attempt < 50; attempt++ {
		path := EndpointFilePath(home)
		if _, err := os.Stat(path); err == nil {
			return readEndpointFile(t, home)
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("endpoint file was not written")
	return EndpointFile{}
}

func readEndpointFile(t *testing.T, home string) EndpointFile {
	t.Helper()
	content, err := os.ReadFile(EndpointFilePath(home))
	if err != nil {
		t.Fatalf("read endpoint: %v", err)
	}
	var endpoint EndpointFile
	if err := json.Unmarshal(content, &endpoint); err != nil {
		t.Fatalf("decode endpoint: %v", err)
	}
	return endpoint
}
