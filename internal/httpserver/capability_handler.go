package httpserver

import (
	"net/http"

	"github.com/spacehz-lab/cal/internal/contract"
)

func (server *Server) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	query := r.URL.Query()
	result, err := server.app.ListCapabilities(r.Context(), &contract.CapabilityListRequest{
		CapabilityID: query.Get(queryCapabilityID),
		ProviderID:   query.Get(queryProviderID),
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
