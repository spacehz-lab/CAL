package control

// APIError is the structured error shape returned by control adapters.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// NewAPIError creates one structured control error.
func NewAPIError(code, message string) APIError {
	return APIError{Code: code, Message: message}
}

func (err APIError) Error() string {
	return err.Message
}
