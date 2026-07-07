package client

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/spacehz-lab/cal/internal/contract"
)

// Error is a structured client-side transport or daemon API error.
type Error struct {
	StatusCode int
	Code       contract.ErrorCode
	Message    string
}

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	if err.Message != "" {
		return err.Message
	}
	if err.Code != "" {
		return string(err.Code)
	}
	return "client error"
}

func decodeErrorResponse(statusCode int, data []byte) error {
	var response contract.ErrorResponse
	if err := json.Unmarshal(data, &response); err == nil && response.Error.Code != "" {
		return newError(statusCode, response.Error.Code, response.Error.Message)
	}
	message := strings.TrimSpace(string(data))
	if message == "" {
		message = http.StatusText(statusCode)
	}
	return newError(statusCode, contract.ErrorInternal, message)
}

func newError(statusCode int, code contract.ErrorCode, message string) *Error {
	if message == "" {
		message = string(code)
	}
	return &Error{StatusCode: statusCode, Code: code, Message: message}
}
