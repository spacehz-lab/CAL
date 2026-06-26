package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/spacehz-lab/cal/internal/cald/control"
)

// RouterConfig wires the HTTP adapter to the control service.
type RouterConfig struct {
	Service control.Service
	Status  control.Status
	Stop    func()
}

// NewRouter builds the local cald HTTP API router.
func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()

	daemon := daemonHandler{status: cfg.Status, stop: cfg.Stop}
	r.Get("/v1/daemon/status", daemon.statusHandler)
	r.Post("/v1/daemon/stop", daemon.stopHandler)

	providers := providerHandler{svc: cfg.Service}
	r.Get("/v1/providers/sources", providers.listSources)
	r.Post("/v1/providers/sources/add", providers.addSource)
	r.Post("/v1/providers/sources/remove", providers.removeSource)
	r.Post("/v1/providers/find", providers.find)
	r.Get("/v1/providers", providers.list)
	r.Get("/v1/providers/{provider_id}", providers.get)

	discovery := discoveryHandler{svc: cfg.Service}
	r.Post("/v1/discovery", discovery.run)

	capabilities := capabilityHandler{svc: cfg.Service}
	r.Get("/v1/capabilities", capabilities.list)
	r.Get("/v1/capabilities/{capability_id}", capabilities.get)

	runs := runHandler{svc: cfg.Service}
	r.Post("/v1/runs", runs.run)
	r.Get("/v1/runs/{run_id}", runs.get)

	uses := useHandler{svc: cfg.Service}
	r.Post("/v1/uses", uses.use)

	eval := evalHandler{svc: cfg.Service}
	r.Get("/v1/eval", eval.metrics)
	r.Get("/v1/traces/{trace_id}", eval.trace)

	return r
}
