package observe

import "context"

// Observer collects observations for one provider kind.
type Observer interface {
	Observe(context.Context, *Request) (*Result, error)
}
