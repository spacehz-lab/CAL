package selector

func selectLocal(candidates []candidate) (*Result, error) {
	if len(candidates) > 1 && candidates[0].score == candidates[1].score && candidates[0].capability.ID != candidates[1].capability.ID {
		return nil, &Error{Code: CodeAmbiguous, Message: "multiple promoted capabilities matched the intent"}
	}
	return resultFromCandidate(
		SourceLocal,
		candidates[0],
		"local intent and input match selected this promoted binding",
		len(candidates),
		nil,
	), nil
}
