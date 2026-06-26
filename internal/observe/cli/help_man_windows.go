//go:build windows

package cli

import (
	"context"
	"fmt"
)

func documentationFallback(_ context.Context, _ string) (DocumentationOutput, error) {
	return DocumentationOutput{}, fmt.Errorf("manual lookup is not supported on windows")
}

func documentationFallbackSource() string {
	return "manual"
}
