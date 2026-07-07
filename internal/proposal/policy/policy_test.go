package policy

import "testing"

func TestValidateAcceptsDefaultPolicy(t *testing.T) {
	if err := Validate(Default()); err != nil {
		t.Fatalf("Validate(Default()) error = %v", err)
	}
}

func TestValidateSurfaceRejectsInvalidPattern(t *testing.T) {
	policy := Default().Surface
	policy.SkipPatterns = []string{"["}

	if err := ValidateSurface(policy); err == nil {
		t.Fatal("ValidateSurface() error = nil, want error")
	}
}
