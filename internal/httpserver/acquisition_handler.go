package httpserver

import (
	"context"
	"net/http"

	"github.com/spacehz-lab/cal/internal/contract"
)

func (server *Server) handleAcquisitions(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var req contract.AcquisitionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	result, err := server.app.Acquire(r.Context(), &req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (server *Server) handleAcquisitionsStream(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var req contract.AcquisitionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	streamResult(w, r, func(ctx context.Context) (*contract.AcquisitionResponse, error) {
		return server.app.Acquire(ctx, &req)
	})
}
