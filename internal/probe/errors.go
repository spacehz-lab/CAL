package probe

import "fmt"

const (
	CodeInvalidProbeInput       = "invalid_probe_input"
	CodeProbeMaterializeFailed  = "probe_materialize_failed"
	CodeVerificationPlanInvalid = "verification_plan_invalid"
	CodeExecutionFailed         = "execution_failed"
	CodeExecutionTimeout        = "execution_timeout"
	CodeVerificationFailed      = "verification_failed"
	CodeUnsupportedVerifyMethod = "unsupported_verify_method"
)

// Error is a coded probe error for orchestration and API mapping.
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
