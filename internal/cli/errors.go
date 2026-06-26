package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/spacehz-lab/cal/internal/cald/control"
)

type commandErrorCode string

const (
	commandErrorCaldStartFailed          commandErrorCode = "cald_start_failed"
	commandErrorCaldUnavailable          commandErrorCode = "cald_unavailable"
	commandErrorInvalidCapabilityInput   commandErrorCode = "invalid_capability_input"
	commandErrorInvalidProviderInput     commandErrorCode = "invalid_provider_input"
	commandErrorInvalidRunInput          commandErrorCode = "invalid_run_input"
	commandErrorInvalidUseInput          commandErrorCode = "invalid_use_input"
	commandErrorInvalidTraceInput        commandErrorCode = "invalid_trace_input"
	commandErrorCapabilityNotFound       commandErrorCode = "capability_not_found"
	commandErrorBindingNotFound          commandErrorCode = "binding_not_found"
	commandErrorInvalidDiscoveryTarget   commandErrorCode = "invalid_discovery_target"
	commandErrorInvalidDiscoveryMode     commandErrorCode = "invalid_discovery_mode"
	commandErrorInvalidDiscoveryProposal commandErrorCode = "invalid_discovery_proposal"
	commandErrorInvalidLLMConfig         commandErrorCode = "invalid_llm_config"
	commandErrorTargetProviderNotFound   commandErrorCode = "target_provider_not_found"
	commandErrorAmbiguousTargetProvider  commandErrorCode = "ambiguous_target_provider"
)

type commandError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func newCommandError(code commandErrorCode, message string) commandError {
	return commandError{Code: string(code), Message: message}
}

func newCommandErrorf(code commandErrorCode, format string, args ...any) commandError {
	return newCommandError(code, fmt.Sprintf(format, args...))
}

func (err commandError) Error() string {
	return err.Message
}

type errorOutput struct {
	Error commandError `json:"error"`
}

func commandErrorFrom(err error) (commandError, bool) {
	var commandErr commandError
	if errors.As(err, &commandErr) {
		return commandErr, true
	}
	var apiErr control.APIError
	if errors.As(err, &apiErr) {
		return commandError{Code: apiErr.Code, Message: apiErr.Message}, true
	}
	return commandError{}, false
}

func writeCommandError(cmd *cobra.Command, jsonOut bool, err error) error {
	if commandErr, ok := commandErrorFrom(err); ok {
		if jsonOut {
			return writeJSONError(cmd.OutOrStdout(), commandErr)
		}
		return commandErr
	}
	return err
}
