package trace

import (
	"fmt"
	"strings"
)

// Validate checks Trace invariants.
func Validate(record Trace) error {
	if strings.TrimSpace(record.ID) == "" {
		return fmt.Errorf("trace id is required")
	}
	switch record.Status {
	case StatusRunning, StatusCompleted, StatusFailed, StatusCanceled:
		return nil
	default:
		return fmt.Errorf("trace status %q is invalid", record.Status)
	}
}
