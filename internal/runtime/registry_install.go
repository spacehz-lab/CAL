package runtime

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spacehz-lab/cal/internal/calpath"
	"github.com/spacehz-lab/cal/internal/core"
)

const (
	generatedVerifierEntry       = "verify.py"
	generatedVerifierRuntime     = "python3"
	generatedVerifierTimeoutMS   = 3000
	maxGeneratedVerifierScriptKB = 32
)

// GeneratedVerifierPackage is a proposal-provided verifier script package.
type GeneratedVerifierPackage struct {
	ID          string `json:"id"`
	Description string `json:"description,omitempty"`
	VerifyPY    string `json:"verify_py"`
}

// InstallVerifier installs a proposal-provided verifier under the configured CAL home.
func InstallVerifier(pkg GeneratedVerifierPackage) error {
	started := time.Now()
	if err := validateGeneratedVerifierPackage(pkg); err != nil {
		logVerifierInstallFailed(pkg.ID, "validate", started)
		return err
	}
	home, err := calpath.HomeDir()
	if err != nil {
		logVerifierInstallFailed(pkg.ID, "home", started)
		return fmt.Errorf("resolve CAL home: %w", err)
	}
	info, err := os.Stat(home)
	if err != nil {
		logVerifierInstallFailed(pkg.ID, "home", started)
		return fmt.Errorf("CAL home %q is not available: %w", home, err)
	}
	if !info.IsDir() {
		logVerifierInstallFailed(pkg.ID, "home", started)
		return fmt.Errorf("CAL home %q is not a directory", home)
	}

	registry := NewRegistry()
	if registry.loadErr != nil {
		logVerifierInstallFailed(pkg.ID, "registry", started)
		return registry.loadErr
	}
	if registry.Supports(pkg.ID) {
		logVerifierInstallFailed(pkg.ID, "exists", started)
		return fmt.Errorf("verifier %q already exists", pkg.ID)
	}

	root := filepath.Join(home, verifierDirName)
	if err := os.MkdirAll(root, 0o755); err != nil {
		logVerifierInstallFailed(pkg.ID, "mkdir", started)
		return fmt.Errorf("create verifier directory: %w", err)
	}
	dir := filepath.Join(root, pkg.ID)
	if _, err := os.Stat(dir); err == nil {
		logVerifierInstallFailed(pkg.ID, "exists", started)
		return fmt.Errorf("verifier %q already exists", pkg.ID)
	} else if !os.IsNotExist(err) {
		logVerifierInstallFailed(pkg.ID, "stat", started)
		return fmt.Errorf("stat generated verifier directory: %w", err)
	}
	if err := os.Mkdir(dir, 0o755); err != nil {
		logVerifierInstallFailed(pkg.ID, "mkdir", started)
		return fmt.Errorf("create generated verifier package: %w", err)
	}
	if err := writeGeneratedVerifierPackage(dir, pkg); err != nil {
		_ = os.RemoveAll(dir)
		logVerifierInstallFailed(pkg.ID, "write", started)
		return err
	}
	slog.Info("runtime verifier installed",
		logKeyVerifierID, pkg.ID,
		logKeyRuntime, generatedVerifierRuntime,
		logKeyTimeoutMS, generatedVerifierTimeoutMS,
		"script_bytes", len(pkg.VerifyPY),
		logKeyDurationMS, time.Since(started).Milliseconds(),
	)
	return nil
}

func validateGeneratedVerifierPackage(pkg GeneratedVerifierPackage) error {
	if !core.ValidVerifierID(pkg.ID) {
		return fmt.Errorf("generated verifier id %q is invalid", pkg.ID)
	}
	if strings.TrimSpace(pkg.VerifyPY) == "" {
		return fmt.Errorf("generated verifier %q verify_py is required", pkg.ID)
	}
	if len(pkg.VerifyPY) > maxGeneratedVerifierScriptKB*1024 {
		return fmt.Errorf("generated verifier %q verify_py exceeds %dKB", pkg.ID, maxGeneratedVerifierScriptKB)
	}
	return nil
}

func writeGeneratedVerifierPackage(dir string, pkg GeneratedVerifierPackage) error {
	meta, err := json.MarshalIndent(scriptVerifier{
		ID:          pkg.ID,
		Description: strings.TrimSpace(pkg.Description),
		Runtime:     generatedVerifierRuntime,
		Entry:       generatedVerifierEntry,
		TimeoutMS:   generatedVerifierTimeoutMS,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode generated verifier metadata: %w", err)
	}
	meta = append(meta, '\n')
	if err := os.WriteFile(filepath.Join(dir, "meta.json"), meta, 0o644); err != nil {
		return fmt.Errorf("write generated verifier metadata: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, generatedVerifierEntry), []byte(pkg.VerifyPY), 0o644); err != nil {
		return fmt.Errorf("write generated verifier script: %w", err)
	}
	return nil
}

func logVerifierInstallFailed(verifierID, stage string, started time.Time) {
	slog.Warn("runtime verifier install failed",
		logKeyVerifierID, verifierID,
		logKeyStage, stage,
		logKeyDurationMS, time.Since(started).Milliseconds(),
	)
}
