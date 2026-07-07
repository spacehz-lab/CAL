package observe

import (
	"context"
	"fmt"

	"github.com/spacehz-lab/cal/internal/model"
)

// Runner dispatches provider observation by provider kind.
type Runner struct {
	observers map[model.ProviderKind]Observer
}

// NewRunner creates an observation runner.
func NewRunner(observers map[model.ProviderKind]Observer) *Runner {
	copied := make(map[model.ProviderKind]Observer, len(observers))
	for kind, observer := range observers {
		copied[kind] = observer
	}
	return &Runner{observers: copied}
}

// Observe collects observations for one provider.
func (runner *Runner) Observe(ctx context.Context, req *Request) (*Result, error) {
	ctx = normalizeContext(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if runner == nil {
		return nil, newError(CodeObserverNotConfigured, "observe runner is not configured")
	}
	if req == nil || req.Provider == nil {
		return nil, newError(CodeInvalidObserveInput, "provider is required")
	}
	if req.Provider.Kind == "" {
		return nil, newError(CodeUnsupportedProviderKind, "provider kind is required")
	}

	observer := runner.observers[req.Provider.Kind]
	if observer == nil {
		return nil, newError(CodeObserverNotConfigured, fmt.Sprintf("observer for provider kind %q is not configured", req.Provider.Kind))
	}

	result, err := observer.Observe(ctx, req)
	if err != nil {
		return nil, wrapError(CodeObservationFailed, "observer failed", err)
	}
	if result == nil {
		return nil, newError(CodeObservationFailed, "observer result is required")
	}
	if result.ProviderID == "" {
		result.ProviderID = req.Provider.ID
	}
	for index := range result.Observations {
		if result.Observations[index].ProviderID == "" {
			result.Observations[index].ProviderID = result.ProviderID
		}
	}
	return result, nil
}

func normalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
