package httpapi

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/spacehz-lab/cal/internal/cald/control"
	"github.com/spacehz-lab/cal/internal/config"
)

type providerHandler struct {
	svc control.Service
}

type sourceRequest struct {
	Kind  config.ProviderSourceKind `json:"kind"`
	Value string                    `json:"value"`
}

func (h providerHandler) listSources(w http.ResponseWriter, _ *http.Request) {
	sources, err := h.svc.ListSources()
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sources": sources})
}

func (h providerHandler) addSource(w http.ResponseWriter, r *http.Request) {
	h.mutateSource(w, r, true)
}

func (h providerHandler) removeSource(w http.ResponseWriter, r *http.Request) {
	h.mutateSource(w, r, false)
}

func (h providerHandler) mutateSource(w http.ResponseWriter, r *http.Request, add bool) {
	var req sourceRequest
	if !decodeRequest(w, r, &req) {
		return
	}
	if req.Kind != config.ProviderSourceKindPath {
		writeError(w, http.StatusUnprocessableEntity, control.NewAPIError("unsupported_provider_source", fmt.Sprintf("provider source kind %q is not supported", req.Kind)))
		return
	}
	var cfg config.Config
	var changed bool
	var err error
	if add {
		cfg, changed, err = h.svc.AddPathSource(req.Value)
	} else {
		cfg, changed, err = h.svc.RemovePathSource(req.Value)
	}
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"changed": changed, "sources": cfg.ProviderSources})
}

func (h providerHandler) find(w http.ResponseWriter, r *http.Request) {
	var req control.ProviderFindRequest
	if !decodeRequest(w, r, &req) {
		return
	}
	result, err := h.svc.FindProviders(r.Context(), req)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h providerHandler) list(w http.ResponseWriter, _ *http.Request) {
	providers, err := h.svc.ListProviders()
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"providers": providers})
}

func (h providerHandler) get(w http.ResponseWriter, r *http.Request) {
	provider, ok, err := h.svc.GetProvider(chi.URLParam(r, "provider_id"))
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeRecord(w, provider, ok)
}
