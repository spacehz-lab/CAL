package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/spacehz-lab/cal/internal/cli/client"
	"github.com/spacehz-lab/cal/internal/contract"
)

const (
	exitGeneral     = 1
	exitUsage       = 2
	exitUnavailable = 3
)

// ExitError carries the process exit code selected by the CLI layer.
type ExitError struct {
	Code int
	Err  error
}

func (err *ExitError) Error() string {
	if err == nil || err.Err == nil {
		return ""
	}
	return err.Err.Error()
}

func (err *ExitError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Err
}

func commandError(cmd *cobra.Command, jsonOut bool, err error) error {
	if err == nil {
		return nil
	}
	exitErr := &ExitError{Code: exitCode(err), Err: err}
	if jsonOut {
		if writeErr := writeJSON(cmd.OutOrStdout(), contract.ErrorResponse{Error: publicError(err)}); writeErr != nil {
			return writeErr
		}
	}
	return exitErr
}

func exitCode(err error) int {
	var clientErr *client.Error
	if errors.As(err, &clientErr) {
		switch clientErr.Code {
		case contract.ErrorInvalidRequest:
			return exitUsage
		case contract.ErrorCaldUnavailable:
			return exitUnavailable
		default:
			return exitGeneral
		}
	}
	return exitGeneral
}

func publicError(err error) contract.Error {
	var clientErr *client.Error
	if errors.As(err, &clientErr) {
		return contract.Error{Code: clientErr.Code, Message: clientErr.Message}
	}
	return contract.Error{Code: contract.ErrorInternal, Message: err.Error()}
}

func invalidInput(format string, args ...any) error {
	return &client.Error{
		Code:    contract.ErrorInvalidRequest,
		Message: fmt.Sprintf(format, args...),
	}
}
