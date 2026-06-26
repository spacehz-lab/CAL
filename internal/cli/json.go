package cli

import (
	"encoding/json"
	"io"
)

func writeJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func writeJSONCommandError(out io.Writer, code commandErrorCode, message string) error {
	return writeJSONError(out, newCommandError(code, message))
}

func writeJSONError(out io.Writer, commandErr commandError) error {
	if err := writeJSON(out, errorOutput{Error: commandErr}); err != nil {
		return err
	}
	return commandErr
}
