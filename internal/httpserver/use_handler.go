package httpserver

import (
	"context"
	"net/http"

	"github.com/spacehz-lab/cal/internal/contract"
)

func (server *Server) handleUses(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var req contract.UseRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	result, err := server.app.Use(r.Context(), &req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (server *Server) handleUsesStream(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var req contract.UseRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	streamResult(w, r, func(ctx context.Context) (*contract.UseResponse, error) {
		return server.app.Use(ctx, &req)
	})
}
