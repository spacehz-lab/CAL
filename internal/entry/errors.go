package entry

import "fmt"

const (
	CodeInvalidProviderPath    = "invalid_provider_path"
	CodeTargetProviderNotFound = "target_provider_not_found"
	CodeProviderNotFound       = "provider_not_found"
	CodeEntryStoreFailed       = "entry_store_failed"
)

// Error is a coded entry error for API and CLI mapping.
type Error struct {
	Code    string
	Message string
	Err     error
}

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	if err.Err != nil {
		return fmt.Sprintf("%s: %s: %v", err.Code, err.Message, err.Err)
	}
	return fmt.Sprintf("%s: %s", err.Code, err.Message)
}

func (err *Error) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Err
}

func newError(code string, message string) error {
	return &Error{Code: code, Message: message}
}

func wrapError(code string, message string, err error) error {
	return &Error{Code: code, Message: message, Err: err}
}
