package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	app "github.com/spacehz-lab/cal/internal/cald/app"
	"github.com/spacehz-lab/cal/internal/contract"
)

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		message := err.Error()
		if errors.Is(err, io.EOF) {
			message = "request body is required"
		}
		writeTransportError(w, http.StatusBadRequest, contract.ErrorInvalidRequest, message)
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeTransportError(w, http.StatusBadRequest, contract.ErrorInvalidRequest, "request body must contain a single JSON value")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeAppError(w http.ResponseWriter, err error) {
	status, code, message := appError(err)
	writeTransportError(w, status, code, message)
}

func appErrorResponse(err error) contract.ErrorResponse {
	_, code, message := appError(err)
	return contract.ErrorResponse{Error: contract.Error{Code: code, Message: message}}
}

func appError(err error) (int, contract.ErrorCode, string) {
	switch {
	case errors.Is(err, app.ErrInvalidMode), errors.Is(err, app.ErrProposalPathRequired):
		return http.StatusBadRequest, contract.ErrorInvalidRequest, err.Error()
	case errors.Is(err, app.ErrLLMNotConfigured), errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return http.StatusServiceUnavailable, contract.ErrorCaldUnavailable, err.Error()
	default:
		return http.StatusInternalServerError, contract.ErrorInternal, err.Error()
	}
}

func writeTransportError(w http.ResponseWriter, status int, code contract.ErrorCode, message string) {
	writeJSON(w, status, contract.ErrorResponse{Error: contract.Error{Code: code, Message: message}})
}
