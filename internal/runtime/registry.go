package runtime

// Registry is kept as the Runner construction boundary while verification is
// handled by built-in VerifySpec checks.
type Registry struct{}

// NewRegistry builds the process verification registry.
func NewRegistry() Registry {
	return Registry{}
}

// DefaultRegistry returns the process verification registry.
func DefaultRegistry() Registry {
	return NewRegistry()
}
