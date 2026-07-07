package promote

import "fmt"

const (
	CodeInvalidPromotionInput = "invalid_promotion_input"
	CodePromotionRejected     = "promotion_rejected"
	CodePromotionStoreFailed  = "promotion_store_failed"
)

const (
	ActionCreated = "created"
	ActionReused  = "reused"
	ActionUpdated = "updated"
)

// Error is a coded promotion error for orchestration and API mapping.
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
