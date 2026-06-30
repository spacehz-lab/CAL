package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var (
	capabilityIDPattern = regexp.MustCompile(`^[a-z0-9]+\.[a-z0-9]+$`)
)

// ValidCapabilityID reports whether id matches the CAL capability id shape.
func ValidCapabilityID(id string) bool {
	return capabilityIDPattern.MatchString(id)
}

// ProviderID derives a deterministic provider id from entry facts.
func ProviderID(platform string, kind ProviderKind, absoluteCleanPath string) string {
	return "provider_" + ShortHash(platform, string(kind), absoluteCleanPath)
}

// BindingID derives a deterministic binding id from a pre-canonicalized
// execution identity. Most callers should use BindingIDForExecution.
func BindingID(capabilityID, providerID, canonicalExecution string) string {
	return "binding_" + ShortHash(capabilityID, providerID, canonicalExecution)
}

// CanonicalExecution returns the stable identity string for one execution plan.
func CanonicalExecution(execution Execution) (string, error) {
	payload, err := json.Marshal(execution)
	if err != nil {
		return "", fmt.Errorf("canonicalize execution: %w", err)
	}
	return string(payload), nil
}

// BindingIDForExecution derives a deterministic binding id from capability,
// provider, and the full canonical execution plan.
func BindingIDForExecution(capabilityID, providerID string, execution Execution) (string, error) {
	canonical, err := CanonicalExecution(execution)
	if err != nil {
		return "", err
	}
	return BindingID(capabilityID, providerID, canonical), nil
}

// ShortHash returns the stable short hash used in local CAL ids.
func ShortHash(parts ...string) string {
	key := strings.Join(parts, "|")
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])[:12]
}
