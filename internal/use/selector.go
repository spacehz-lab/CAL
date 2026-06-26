package use

import "context"

type selector interface {
	selectBinding(context.Context, Request, []candidate) (Selection, *Error)
}

type localSelector struct{}

func (localSelector) selectBinding(_ context.Context, _ Request, candidates []candidate) (Selection, *Error) {
	if len(candidates) > 1 && candidates[0].score == candidates[1].score && candidates[0].capability.ID != candidates[1].capability.ID {
		return Selection{}, &Error{Code: CodeAmbiguous, Message: "multiple promoted capabilities matched the intent"}
	}
	best := candidates[0]
	return Selection{
		Source:       selectionSourceLocal,
		CapabilityID: best.capability.ID,
		BindingID:    best.binding.ID,
		ProviderID:   best.binding.ProviderID,
		Reason:       "local intent and input match selected this promoted binding",
	}, nil
}
