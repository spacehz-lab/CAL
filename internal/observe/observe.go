package observe

import (
	"context"

	"github.com/spacehz-lab/cal/internal/core"
)

// Observer collects observations from one provider entry.
type Observer interface {
	Observe(context.Context, core.Provider) (Result, error)
}
