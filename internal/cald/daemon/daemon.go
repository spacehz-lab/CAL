package daemon

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	app "github.com/spacehz-lab/cal/internal/cald/app"
	"github.com/spacehz-lab/cal/internal/cald/endpoint"
	"github.com/spacehz-lab/cal/internal/contract"
	"github.com/spacehz-lab/cal/internal/httpserver"
	"github.com/spacehz-lab/cal/internal/logging"
)

const (
	defaultAddr            = "127.0.0.1:0"
	defaultShutdownTimeout = 3 * time.Second
	defaultLogName         = "cald"
)

var ErrHomeRequired = errors.New("daemon app home is required")

// Options configures one local cald daemon instance.
type Options struct {
	Home            string
	WorkRoot        string
	Addr            string
	Logging         logging.Options
	ShutdownTimeout time.Duration
	Now             func() time.Time
}

// Daemon owns local process lifecycle for one cald HTTP server.
type Daemon struct {
	options         Options
	shutdownTimeout time.Duration
	now             func() time.Time
}

// New creates a local cald daemon.
func New(opts Options) (*Daemon, error) {
	if strings.TrimSpace(opts.Home) == "" {
		return nil, ErrHomeRequired
	}
	opts.Home = strings.TrimSpace(opts.Home)
	if strings.TrimSpace(opts.Addr) == "" {
		opts.Addr = defaultAddr
	}
	shutdownTimeout := opts.ShutdownTimeout
	if shutdownTimeout <= 0 {
		shutdownTimeout = defaultShutdownTimeout
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	return &Daemon{options: opts, shutdownTimeout: shutdownTimeout, now: now}, nil
}

// Serve starts the local cald HTTP server and blocks until shutdown.
func (daemon *Daemon) Serve(ctx context.Context) error {
	if daemon == nil {
		return ErrHomeRequired
	}
	ctx = normalizeContext(ctx)
	daemon.configureLogging()

	application, err := app.New(app.Options{
		Home:     daemon.options.Home,
		WorkRoot: daemon.options.WorkRoot,
		Now:      daemon.now,
	})
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", daemon.options.Addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer listener.Close()

	status := daemon.status(listener)
	if err := endpoint.Write(daemon.options.Home, daemon.endpointRecord(status)); err != nil {
		return err
	}
	defer endpoint.Remove(daemon.options.Home)

	server := &http.Server{}
	var shutdownOnce sync.Once
	shutdown := func() {
		shutdownOnce.Do(func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), daemon.shutdownTimeout)
			defer cancel()
			_ = server.Shutdown(shutdownCtx)
		})
	}

	handler, err := httpserver.New(httpserver.Options{
		App: application,
		Daemon: httpserver.DaemonControl{
			Status: func() contract.DaemonStatus {
				return status
			},
			Stop: shutdown,
		},
	})
	if err != nil {
		return err
	}
	server.Handler = handler

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		shutdown()
		<-errCh
		return ctx.Err()
	case err := <-errCh:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func (daemon *Daemon) status(listener net.Listener) contract.DaemonStatus {
	return contract.DaemonStatus{
		Running: true,
		BaseURL: "http://" + listener.Addr().String(),
		PID:     os.Getpid(),
	}
}

func (daemon *Daemon) endpointRecord(status contract.DaemonStatus) *endpoint.Record {
	return &endpoint.Record{
		BaseURL:   status.BaseURL,
		PID:       status.PID,
		CreatedAt: daemon.now().UTC().Format(time.RFC3339Nano),
	}
}

func (daemon *Daemon) configureLogging() {
	if !loggingConfigured(daemon.options.Logging) {
		return
	}
	opts := daemon.options.Logging
	if strings.TrimSpace(opts.Name) == "" {
		opts.Name = defaultLogName
	}
	logging.Configure(&opts)
}

func normalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func loggingConfigured(opts logging.Options) bool {
	return strings.TrimSpace(opts.Name) != "" ||
		opts.Err != nil ||
		opts.Env != nil ||
		strings.TrimSpace(string(opts.Config.Level)) != "" ||
		opts.Config.File.Enabled != nil ||
		opts.Config.File.MaxBytes != 0 ||
		opts.Config.File.MaxFiles != 0
}
