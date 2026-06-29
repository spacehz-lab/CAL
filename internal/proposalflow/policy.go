package proposalflow

import (
	"fmt"
	"regexp"
	"strings"
)

const PolicyFileName = "proposal_policy.json"

// Policy controls local Proposal normalization and filtering.
type Policy struct {
	Surface    SurfacePolicy    `json:"surface"`
	Capability CapabilityPolicy `json:"capability"`
}

// SurfacePolicy controls Stage1 Surface material accepted for Capability planning.
type SurfacePolicy struct {
	AllowedKinds []string `json:"allowed_kinds"`
	SkipNames    []string `json:"skip_names"`
	SkipPatterns []string `json:"skip_patterns"`
}

// CapabilityPolicy is reserved for Stage2 capability-id policy.
type CapabilityPolicy struct {
	AllowedSubjects []string `json:"allowed_subjects"`
	BlockedSubjects []string `json:"blocked_subjects"`
}

// DefaultPolicy returns the complete default Proposal policy.
func DefaultPolicy() Policy {
	return Policy{
		Surface: SurfacePolicy{
			AllowedKinds: []string{"command", "subcommand", "mode", "option"},
			SkipNames:    []string{"help", "version", "usage"},
			SkipPatterns: []string{},
		},
		Capability: CapabilityPolicy{
			AllowedSubjects: []string{},
			BlockedSubjects: []string{},
		},
	}
}

// ValidatePolicy checks that a complete Proposal policy can be applied locally.
func ValidatePolicy(policy Policy) error {
	if len(policy.Surface.AllowedKinds) == 0 {
		return fmt.Errorf("proposal surface allowed_kinds is required")
	}
	seen := map[string]struct{}{}
	for _, kind := range policy.Surface.AllowedKinds {
		kind = normalizePolicyToken(kind)
		if !validSurfaceKind(kind) {
			return fmt.Errorf("unsupported proposal surface kind %q", kind)
		}
		if _, ok := seen[kind]; ok {
			return fmt.Errorf("duplicate proposal surface kind %q", kind)
		}
		seen[kind] = struct{}{}
	}
	for _, pattern := range policy.Surface.SkipPatterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("invalid proposal surface skip pattern %q: %w", pattern, err)
		}
	}
	return nil
}

func normalizePolicyToken(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func validSurfaceKind(kind string) bool {
	switch kind {
	case "command", "subcommand", "mode", "option":
		return true
	default:
		return false
	}
}
