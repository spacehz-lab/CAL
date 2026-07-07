package httpserver

import (
	"context"
	"net/http"

	"github.com/spacehz-lab/cal/internal/contract"
)

func (server *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var req contract.RunRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	result, err := server.app.Run(r.Context(), &req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (server *Server) handleRunsStream(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var req contract.RunRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	streamResult(w, r, func(ctx context.Context) (*contract.RunResponse, error) {
		return server.app.Run(ctx, &req)
	})
}
