package httpserver

import (
	"net/http"

	"github.com/spacehz-lab/cal/internal/contract"
)

func (server *Server) handleEval(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	query := r.URL.Query()
	result, err := server.app.Eval(r.Context(), &contract.EvalRequest{
		ProviderID:   query.Get(queryProviderID),
		CapabilityID: query.Get(queryCapabilityID),
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
