package cald

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spacehz-lab/cal/internal/cald/control"
	"github.com/spacehz-lab/cal/internal/cald/httpapi"
)

func serve(ctx context.Context, home string) error {
	svc, err := control.NewService(home)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer listener.Close()

	started := time.Now().UTC()
	baseURL := "http://" + listener.Addr().String()
	status := Status{
		Running:   true,
		Mode:      "local",
		PID:       os.Getpid(),
		Endpoint:  baseURL,
		StartedAt: started.Format(time.RFC3339Nano),
	}
	if err := writeConnectionFiles(svc.Home(), status); err != nil {
		return err
	}
	defer os.Remove(EndpointFilePath(svc.Home()))

	server := &http.Server{}
	server.Handler = httpapi.NewRouter(httpapi.RouterConfig{
		Service: svc,
		Status:  status,
		Stop: func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = server.Shutdown(shutdownCtx)
		},
	})
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
		return ctx.Err()
	case err := <-errCh:
		if err == nil || err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

func writeConnectionFiles(home string, status Status) error {
	path := EndpointFilePath(home)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create cald connection directory: %w", err)
	}
	if err := writeJSONFile(path, EndpointFile{
		PID:       status.PID,
		BaseURL:   status.Endpoint,
		StartedAt: status.StartedAt,
	}); err != nil {
		return err
	}
	return nil
}

func writeJSONFile(path string, value any) error {
	temp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*.json")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)

	encoder := json.NewEncoder(temp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		temp.Close()
		return fmt.Errorf("encode json file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close json file: %w", err)
	}
	if err := os.Rename(tempName, path); err != nil {
		return fmt.Errorf("rename json file: %w", err)
	}
	return nil
}
