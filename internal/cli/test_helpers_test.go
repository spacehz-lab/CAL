package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"

	"github.com/spacehz-lab/cal/internal/cald/endpoint"
)

func newTestCLI(t *testing.T, handler http.HandlerFunc) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	home := t.TempDir()
	if handler != nil {
		server := httptest.NewServer(handler)
		t.Cleanup(server.Close)
		if err := endpoint.Write(home, &endpoint.Record{BaseURL: server.URL, PID: 123}); err != nil {
			t.Fatalf("endpoint.Write() error = %v", err)
		}
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	app, err := New(Options{Home: home, Stdout: stdout, Stderr: stderr, Environ: []string{}})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return app.Command(), stdout, stderr
}

func execute(t *testing.T, cmd *cobra.Command, args ...string) error {
	t.Helper()
	cmd.SetArgs(args)
	return cmd.ExecuteContext(context.Background())
}

func decodeOutput[T any](t *testing.T, out *bytes.Buffer) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(out.Bytes(), &value); err != nil {
		t.Fatalf("decode output: %v; output = %s", err, out.String())
	}
	return value
}

func writeResponse(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
}

func decodeRequest[T any](t *testing.T, r *http.Request) T {
	t.Helper()
	var value T
	if err := json.NewDecoder(r.Body).Decode(&value); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	return value
}
