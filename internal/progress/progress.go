package progress

import (
	"context"

	"github.com/spacehz-lab/cal/internal/model"
)

// Handler observes one live workflow progress event.
type Handler func(context.Context, *model.ProgressEvent)

type handlerKey struct{}

// WithHandler attaches one request-local progress handler to a context.
func WithHandler(ctx context.Context, handler Handler) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if handler == nil {
		return ctx
	}
	if existing, ok := ctx.Value(handlerKey{}).(Handler); ok && existing != nil {
		handler = chain(existing, handler)
	}
	return context.WithValue(ctx, handlerKey{}, handler)
}

// Emit sends one progress event to the request-local handler and explicit handlers.
func Emit(ctx context.Context, event *model.ProgressEvent, handlers ...Handler) {
	if event == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if handler, ok := ctx.Value(handlerKey{}).(Handler); ok && handler != nil {
		handler(ctx, event)
	}
	for _, handler := range handlers {
		if handler != nil {
			handler(ctx, event)
		}
	}
}

func chain(first Handler, second Handler) Handler {
	return func(ctx context.Context, event *model.ProgressEvent) {
		first(ctx, event)
		second(ctx, event)
	}
}
