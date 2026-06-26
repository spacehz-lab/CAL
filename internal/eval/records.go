package eval

import (
	"github.com/spacehz-lab/cal/internal/core"
	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

type records struct {
	providers    []core.Provider
	capabilities []core.Capability
	runs         []core.Run
	traces       []caltrace.Trace
}
