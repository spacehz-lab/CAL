package proposal

import (
	"fmt"
	"regexp"
	"strings"
)

const PolicyFileName = "proposal_policy.json"

var capabilityTermPattern = regexp.MustCompile(`^[a-z][a-z0-9]*$`)

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

// CapabilityPolicy controls Stage2 capability-id planning ranges.
type CapabilityPolicy struct {
	PreferredSubjects   []string `json:"preferred_subjects"`
	PreferredOperations []string `json:"preferred_operations"`
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
			PreferredSubjects: []string{
				"file",
				"directory",
				"path",
				"text",
				"bytes",
				"json",
				"xml",
				"csv",
				"yaml",
				"document",
				"pdf",
				"image",
				"audio",
				"video",
				"archive",
				"package",
				"source",
				"repository",
				"database",
				"table",
				"http",
				"url",
				"network",
				"process",
				"system",
				"environment",
				"key",
				"certificate",
				"container",
				"project",
			},
			PreferredOperations: []string{
				"read",
				"write",
				"copy",
				"move",
				"remove",
				"list",
				"find",
				"inspect",
				"identify",
				"count",
				"search",
				"filter",
				"query",
				"extract",
				"transform",
				"convert",
				"render",
				"encode",
				"decode",
				"compress",
				"decompress",
				"create",
				"checksum",
				"sign",
				"verify",
				"encrypt",
				"decrypt",
				"download",
				"upload",
				"request",
				"sync",
				"install",
				"update",
				"build",
				"test",
				"run",
				"format",
				"validate",
				"sort",
				"deduplicate",
				"compare",
			},
		},
	}
}

// ValidatePolicy checks that a complete Proposal policy can be applied locally.
func ValidatePolicy(policy Policy) error {
	if err := validateSurfacePolicy(policy.Surface); err != nil {
		return err
	}
	if err := validateCapabilityPolicy(policy.Capability); err != nil {
		return err
	}
	return nil
}

func validateSurfacePolicy(policy SurfacePolicy) error {
	if len(policy.AllowedKinds) == 0 {
		return fmt.Errorf("proposal surface allowed_kinds is required")
	}
	seen := map[string]struct{}{}
	for _, kind := range policy.AllowedKinds {
		kind = normalizePolicyToken(kind)
		if !validSurfaceKind(kind) {
			return fmt.Errorf("unsupported proposal surface kind %q", kind)
		}
		if _, ok := seen[kind]; ok {
			return fmt.Errorf("duplicate proposal surface kind %q", kind)
		}
		seen[kind] = struct{}{}
	}
	for _, pattern := range policy.SkipPatterns {
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

func validateCapabilityPolicy(policy CapabilityPolicy) error {
	if len(policy.PreferredSubjects) == 0 {
		return fmt.Errorf("proposal capability preferred_subjects is required")
	}
	if len(policy.PreferredOperations) == 0 {
		return fmt.Errorf("proposal capability preferred_operations is required")
	}
	if err := validateCapabilityTerms("subject", policy.PreferredSubjects); err != nil {
		return err
	}
	if err := validateCapabilityTerms("operation", policy.PreferredOperations); err != nil {
		return err
	}
	return nil
}

func validateCapabilityTerms(name string, terms []string) error {
	seen := map[string]struct{}{}
	for _, term := range terms {
		term = normalizePolicyToken(term)
		if !capabilityTermPattern.MatchString(term) {
			return fmt.Errorf("invalid proposal capability %s %q", name, term)
		}
		if _, ok := seen[term]; ok {
			return fmt.Errorf("duplicate proposal capability %s %q", name, term)
		}
		seen[term] = struct{}{}
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
