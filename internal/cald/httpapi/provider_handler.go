package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/spacehz-lab/cal/internal/cald/control"
)

type providerHandler struct {
	svc control.Service
}

func (h providerHandler) add(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProviderPath string `json:"provider_path"`
	}
	if !decodeRequest(w, r, &req) {
		return
	}
	provider, err := h.svc.AddProvider(r.Context(), req.ProviderPath)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, provider)
}

func (h providerHandler) list(w http.ResponseWriter, _ *http.Request) {
	providers, err := h.svc.ListProviders()
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"providers": providers})
}

func (h providerHandler) getByPath(w http.ResponseWriter, r *http.Request) {
	provider, ok, err := h.svc.GetProviderByPath(r.URL.Query().Get("provider_path"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeRecord(w, provider, ok)
}

func (h providerHandler) get(w http.ResponseWriter, r *http.Request) {
	provider, ok, err := h.svc.GetProvider(chi.URLParam(r, "provider_id"))
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeRecord(w, provider, ok)
}
