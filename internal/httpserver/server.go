package httpserver

import (
	"errors"
	"net/http"

	app "github.com/spacehz-lab/cal/internal/cald/app"
	"github.com/spacehz-lab/cal/internal/contract"
)

const (
	routeDaemonStatus       = "/v1/daemon/status"
	routeDaemonStop         = "/v1/daemon/stop"
	routeProviders          = "/v1/providers"
	routeCapabilities       = "/v1/capabilities"
	routeAcquisitions       = "/v1/acquisitions"
	routeAcquisitionsStream = "/v1/acquisitions/stream"
	routeRuns               = "/v1/runs"
	routeRunsStream         = "/v1/runs/stream"
	routeUses               = "/v1/uses"
	routeUsesStream         = "/v1/uses/stream"
	routeEval               = "/v1/eval"
)

const (
	queryCapabilityID = "capability_id"
	queryProviderID   = "provider_id"
)

var ErrMissingApp = errors.New("httpserver app is required")

// Options configures one local cald HTTP server adapter.
type Options struct {
	App    *app.App
	Daemon DaemonControl
}

// DaemonControl adapts daemon lifecycle state into HTTP endpoints.
type DaemonControl struct {
	Status func() contract.DaemonStatus
	Stop   func()
}

// Server adapts local HTTP requests to cald application methods.
type Server struct {
	app    *app.App
	daemon DaemonControl
	mux    *http.ServeMux
}

// New creates a local cald HTTP handler.
func New(opts Options) (*Server, error) {
	if opts.App == nil {
		return nil, ErrMissingApp
	}
	server := &Server{
		app:    opts.App,
		daemon: opts.Daemon,
		mux:    http.NewServeMux(),
	}
	server.routes()
	return server, nil
}

// ServeHTTP routes one local daemon HTTP request.
func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if server == nil || server.mux == nil {
		writeTransportError(w, http.StatusServiceUnavailable, contract.ErrorCaldUnavailable, "http server is not configured")
		return
	}
	server.mux.ServeHTTP(w, r)
}

func (server *Server) routes() {
	server.mux.HandleFunc(routeDaemonStatus, server.handleDaemonStatus)
	server.mux.HandleFunc(routeDaemonStop, server.handleDaemonStop)
	server.mux.HandleFunc(routeProviders, server.handleProviders)
	server.mux.HandleFunc(routeCapabilities, server.handleCapabilities)
	server.mux.HandleFunc(routeAcquisitions, server.handleAcquisitions)
	server.mux.HandleFunc(routeAcquisitionsStream, server.handleAcquisitionsStream)
	server.mux.HandleFunc(routeRuns, server.handleRuns)
	server.mux.HandleFunc(routeRunsStream, server.handleRunsStream)
	server.mux.HandleFunc(routeUses, server.handleUses)
	server.mux.HandleFunc(routeUsesStream, server.handleUsesStream)
	server.mux.HandleFunc(routeEval, server.handleEval)
	server.mux.HandleFunc("/", server.handleNotFound)
}

func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	w.Header().Set("Allow", method)
	writeTransportError(w, http.StatusMethodNotAllowed, contract.ErrorInvalidRequest, "method not allowed")
	return false
}
