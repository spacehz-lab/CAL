//go:build windows

package cli

import (
	"context"
	"fmt"
)

func usageFallback(_ context.Context, _ string) (UsageOutput, error) {
	return UsageOutput{}, fmt.Errorf("manual lookup is not supported on windows")
}
