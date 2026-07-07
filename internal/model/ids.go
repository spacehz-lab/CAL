package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	capabilityIDPatternExpr = `^[a-z0-9]+\.[a-z0-9]+$`

	idPrefixProvider = "provider_"
	idPrefixBinding  = "binding_"
	idPrefixTrace    = "trace_"
	idPrefixRun      = "run_"

	idHashSeparator = "|"
	idHashLength    = 12
)

var capabilityIDPattern = regexp.MustCompile(capabilityIDPatternExpr)

// ValidCapabilityID reports whether id matches the CAL capability id shape.
func ValidCapabilityID(id string) bool {
	return capabilityIDPattern.MatchString(id)
}

// ProviderID derives a deterministic provider id from entry facts.
func ProviderID(platform string, kind ProviderKind, absoluteCleanPath string) string {
	return idPrefixProvider + ShortHash(platform, string(kind), absoluteCleanPath)
}

// BindingID derives a deterministic binding id from a pre-canonicalized
// execution identity. Most callers should use BindingIDForExecution.
func BindingID(capabilityID, providerID, canonicalExecution string) string {
	return idPrefixBinding + ShortHash(capabilityID, providerID, canonicalExecution)
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

// TraceID derives a local trace id from a timestamp.
func TraceID(now time.Time) string {
	return idPrefixTrace + strconv.FormatInt(now.UTC().UnixNano(), 10)
}

// RunID derives a local run id from a timestamp.
func RunID(now time.Time) string {
	return idPrefixRun + strconv.FormatInt(now.UTC().UnixNano(), 10)
}

// ShortHash returns the stable short hash used in local CAL ids.
func ShortHash(parts ...string) string {
	key := strings.Join(parts, idHashSeparator)
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])[:idHashLength]
}
