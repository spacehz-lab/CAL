package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/spacehz-lab/cal/internal/calpath"
	"github.com/spacehz-lab/cal/internal/core"
)

const (
	verifierDirName = "verifiers"

	logKeyVerifierID     = "verifier_id"
	logKeyVerifierSource = "verifier_source"
	logKeyRuntime        = "runtime"
	logKeyTimeoutMS      = "timeout_ms"
	logKeyStage          = "stage"
	logKeyDurationMS     = "duration_ms"
)

// Registry dispatches deterministic verifier packages by id.
type Registry struct {
	verifiers map[string]scriptVerifier
	loadErr   error
}

type scriptVerifier struct {
	ID          string `json:"id"`
	Description string `json:"description,omitempty"`
	Runtime     string `json:"runtime"`
	Entry       string `json:"entry"`
	TimeoutMS   int    `json:"timeout_ms"`
	packageDir  string
}

type verifierRequest struct {
	Verifier core.Verifier  `json:"verifier"`
	Inputs   map[string]any `json:"inputs"`
}

type verifierResult struct {
	Passed   bool               `json:"passed"`
	Evidence []core.EvidenceRef `json:"evidence,omitempty"`
	Outputs  map[string]any     `json:"outputs,omitempty"`
	Error    *core.RecordError  `json:"error,omitempty"`
}

// NewRegistry builds a registry from the configured CAL home verifiers directory.
func NewRegistry() Registry {
	registry := Registry{verifiers: map[string]scriptVerifier{}}
	home, err := calpath.HomeDir()
	if err != nil {
		registry.loadErr = err
		return registry
	}
	if err := registry.LoadScriptVerifiers(filepath.Join(home, verifierDirName)); err != nil && !os.IsNotExist(err) {
		registry.loadErr = joinVerifierError(registry.loadErr, err)
	}
	return registry
}

// DefaultRegistry returns the process verifier registry.
func DefaultRegistry() Registry {
	return NewRegistry()
}

// LoadScriptVerifiers registers script verifiers from dir.
func (registry *Registry) LoadScriptVerifiers(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		verifierDir := filepath.Join(dir, entry.Name())
		verifier, err := readScriptVerifier(os.DirFS(verifierDir), ".", entry.Name())
		if err != nil {
			return err
		}
		verifier.packageDir = verifierDir
		registry.add(verifier)
	}
	return nil
}

// Supports reports whether the registry can execute a verifier package id.
func (registry Registry) Supports(verifierID string) bool {
	_, ok := registry.verifiers[verifierID]
	return ok
}

// Verify checks a deterministic verifier after execution.
func (registry Registry) Verify(ctx context.Context, verifier core.Verifier, inputs map[string]any) ([]core.EvidenceRef, map[string]any, error) {
	started := time.Now()
	if registry.loadErr != nil {
		registry.logVerifierFailed(verifier.ID, "", "load", started)
		return nil, nil, registry.loadErr
	}
	script, ok := registry.verifiers[verifier.ID]
	if !ok {
		registry.logVerifierFailed(verifier.ID, "", "lookup", started)
		return nil, nil, fmt.Errorf("verifier %q is not supported", verifier.ID)
	}
	slog.Info("runtime verifier started",
		logKeyVerifierID, verifier.ID,
		logKeyVerifierSource, script.source(),
		logKeyRuntime, script.Runtime,
		logKeyTimeoutMS, script.TimeoutMS,
	)
	evidence, outputs, err := script.verify(ctx, verifier, inputs)
	if err != nil {
		registry.logVerifierFailed(verifier.ID, script.source(), "execute", started)
		return nil, nil, err
	}
	slog.Info("runtime verifier completed",
		logKeyVerifierID, verifier.ID,
		logKeyVerifierSource, script.source(),
		"evidence_count", len(evidence),
		"output_key_count", len(outputs),
		logKeyDurationMS, time.Since(started).Milliseconds(),
	)
	return evidence, outputs, nil
}

func (registry *Registry) add(verifier scriptVerifier) {
	if _, exists := registry.verifiers[verifier.ID]; exists {
		return
	}
	registry.verifiers[verifier.ID] = verifier
}

func readScriptVerifier(fsys fs.FS, dir, fallbackType string) (scriptVerifier, error) {
	content, err := fs.ReadFile(fsys, path.Join(dir, "meta.json"))
	if err != nil {
		return scriptVerifier{}, fmt.Errorf("read verifier metadata: %w", err)
	}
	var verifier scriptVerifier
	if err := json.Unmarshal(content, &verifier); err != nil {
		return scriptVerifier{}, fmt.Errorf("decode verifier metadata: %w", err)
	}
	if err := validateScriptVerifier(verifier, fallbackType); err != nil {
		return scriptVerifier{}, err
	}
	verifier.packageDir = dir
	return verifier, nil
}

func validateScriptVerifier(verifier scriptVerifier, fallbackType string) error {
	if verifier.ID == "" {
		return fmt.Errorf("verifier metadata id is required")
	}
	if !core.ValidVerifierID(verifier.ID) {
		return fmt.Errorf("verifier metadata id %q is invalid", verifier.ID)
	}
	if verifier.ID != fallbackType {
		return fmt.Errorf("verifier metadata id %q does not match directory %q", verifier.ID, fallbackType)
	}
	if verifier.Runtime == "" {
		return fmt.Errorf("verifier %q runtime is required", verifier.ID)
	}
	if verifier.Entry == "" {
		return fmt.Errorf("verifier %q entry is required", verifier.ID)
	}
	cleanEntry := filepath.Clean(verifier.Entry)
	if filepath.IsAbs(verifier.Entry) || cleanEntry == "." || cleanEntry == ".." || strings.HasPrefix(cleanEntry, ".."+string(filepath.Separator)) {
		return fmt.Errorf("verifier %q entry must stay inside its verifier directory", verifier.ID)
	}
	if verifier.TimeoutMS <= 0 {
		return fmt.Errorf("verifier %q timeout_ms must be positive", verifier.ID)
	}
	return nil
}

func (verifier scriptVerifier) verify(ctx context.Context, spec core.Verifier, inputs map[string]any) ([]core.EvidenceRef, map[string]any, error) {
	script, cleanup, err := verifier.entryPath()
	if err != nil {
		return nil, nil, err
	}
	defer cleanup()

	timeout := time.Duration(verifier.TimeoutMS) * time.Millisecond
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	request, err := json.Marshal(verifierRequest{Verifier: spec, Inputs: inputs})
	if err != nil {
		return nil, nil, fmt.Errorf("encode verifier request: %w", err)
	}
	runtimePath := verifierRuntime(verifier.Runtime)
	cmd := exec.CommandContext(runCtx, runtimePath, script)
	cmd.Stdin = bytes.NewReader(request)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if runCtx.Err() == context.DeadlineExceeded {
		return nil, nil, fmt.Errorf("verifier %q timed out after %s", verifier.ID, timeout)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("verifier %q failed: %w%s", verifier.ID, err, stderrSuffix(stderr.String()))
	}

	var result verifierResult
	decoder := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
	decoder.UseNumber()
	if err := decoder.Decode(&result); err != nil {
		return nil, nil, fmt.Errorf("decode verifier %q result: %w", verifier.ID, err)
	}
	if !result.Passed {
		if result.Error != nil && result.Error.Message != "" {
			return nil, nil, fmt.Errorf("%s", result.Error.Message)
		}
		return nil, nil, fmt.Errorf("verifier %q did not pass", verifier.ID)
	}
	result.Outputs = normalizeJSONMap(result.Outputs)
	for index := range result.Evidence {
		result.Evidence[index].Content = normalizeJSONMap(result.Evidence[index].Content)
	}
	return result.Evidence, result.Outputs, nil
}

func (verifier scriptVerifier) entryPath() (string, func(), error) {
	return filepath.Join(verifier.packageDir, verifier.Entry), func() {}, nil
}

func (verifier scriptVerifier) source() string {
	return "local"
}

func (registry Registry) logVerifierFailed(verifierID, source, stage string, started time.Time) {
	slog.Warn("runtime verifier failed",
		logKeyVerifierID, verifierID,
		logKeyVerifierSource, source,
		logKeyStage, stage,
		logKeyDurationMS, time.Since(started).Milliseconds(),
	)
}

func stderrSuffix(stderr string) string {
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return ""
	}
	return ": " + stderr
}

func verifierRuntime(runtime string) string {
	if runtime != "python3" {
		return runtime
	}
	if path, err := exec.LookPath(runtime); err == nil {
		return path
	}
	return runtime
}

func joinVerifierError(first, second error) error {
	if first == nil {
		return second
	}
	return fmt.Errorf("%v; %w", first, second)
}

func normalizeJSONMap(values map[string]any) map[string]any {
	for key, value := range values {
		values[key] = normalizeJSONValue(value)
	}
	return values
}

func normalizeJSONValue(value any) any {
	switch typed := value.(type) {
	case json.Number:
		if intValue, err := typed.Int64(); err == nil {
			return int(intValue)
		}
		if floatValue, err := typed.Float64(); err == nil {
			return floatValue
		}
		return typed.String()
	case map[string]any:
		return normalizeJSONMap(typed)
	case []any:
		for index, item := range typed {
			typed[index] = normalizeJSONValue(item)
		}
		return typed
	default:
		return value
	}
}
