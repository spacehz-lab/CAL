package contract

// ErrorCode identifies a stable public HTTP/CLI error category.
type ErrorCode string

const (
	ErrorInvalidRequest  ErrorCode = "invalid_request"
	ErrorNotFound        ErrorCode = "not_found"
	ErrorInternal        ErrorCode = "internal_error"
	ErrorCaldUnavailable ErrorCode = "cald_unavailable"
)

// Error is the stable error payload returned by daemon-facing transports.
type Error struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

// ErrorResponse wraps a structured transport error.
type ErrorResponse struct {
	Error Error `json:"error"`
}
