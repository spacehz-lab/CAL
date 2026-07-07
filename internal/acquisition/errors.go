package acquisition

import "fmt"

const (
	CodeInvalidAcquisitionInput = "invalid_acquisition_input"
	CodeProviderLoadFailed      = "provider_load_failed"
	CodeCatalogLoadFailed       = "catalog_load_failed"
	CodeObserveFailed           = "observe_failed"
	CodeProposalFailed          = "proposal_failed"
	CodeProbeFailed             = "probe_failed"
	CodePromotionFailed         = "promotion_failed"
	CodeTraceWriteFailed        = "trace_write_failed"
)

// Error is a coded acquisition error for orchestration and API mapping.
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
