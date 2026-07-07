package httpserver

import (
	"net/http"

	"github.com/spacehz-lab/cal/internal/contract"
)

func (server *Server) handleProviders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		server.listProviders(w, r)
	case http.MethodPost:
		server.addProvider(w, r)
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		writeTransportError(w, http.StatusMethodNotAllowed, contract.ErrorInvalidRequest, "method not allowed")
	}
}

func (server *Server) listProviders(w http.ResponseWriter, r *http.Request) {
	result, err := server.app.ListProviders(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (server *Server) addProvider(w http.ResponseWriter, r *http.Request) {
	var req contract.AddProviderRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	result, err := server.app.AddProvider(r.Context(), &req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
