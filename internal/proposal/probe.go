package proposal

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spacehz-lab/cal/internal/core"
	"github.com/spacehz-lab/cal/internal/runtime"
)

type probeMaterializer struct {
	workDir          string
	proposalHash     string
	candidateIndex   int
	probePlanIndex   int
	verifierPackages []runtime.GeneratedVerifierPackage
}

func newProbeMaterializer(workDir string, proposalHash string, candidateIndex int, probePlanIndex int, verifierPackages []runtime.GeneratedVerifierPackage) probeMaterializer {
	return probeMaterializer{
		workDir:          workDir,
		proposalHash:     proposalHash,
		candidateIndex:   candidateIndex,
		probePlanIndex:   probePlanIndex,
		verifierPackages: verifierPackages,
	}
}

func (probe probeMaterializer) materialize(plan ProbePlanSpec) (ProbePlan, error) {
	inputs, err := probe.materializeInputs(plan.Inputs)
	if err != nil {
		return ProbePlan{}, err
	}
	for _, fixture := range plan.Fixtures {
		path, err := probe.materializeFixture(fixture)
		if err != nil {
			return ProbePlan{}, err
		}
		inputs[fixture.Input] = path
	}
	verifier, err := probe.ensureVerifier(plan.Verifier)
	if err != nil {
		return ProbePlan{}, err
	}
	return ProbePlan{Inputs: inputs, Verifier: verifier}, nil
}

func (probe probeMaterializer) materializeInputs(inputs map[string]any) (map[string]any, error) {
	materialized := make(map[string]any, len(inputs))
	for key, value := range inputs {
		text, ok := value.(string)
		if !ok {
			materialized[key] = value
			continue
		}
		rendered, err := probe.materializeInputString(key, text)
		if err != nil {
			return nil, err
		}
		materialized[key] = rendered
	}
	return materialized, nil
}

func (probe probeMaterializer) materializeInputString(key, text string) (string, error) {
	rendered := strings.ReplaceAll(text, "{{workdir}}", probe.workDir)
	if strings.Contains(rendered, "{{") || strings.Contains(rendered, "}}") {
		return "", fmt.Errorf("proposal input %q has unresolved template", key)
	}
	if filepath.IsAbs(rendered) {
		if !probe.isWithinWorkDir(rendered) {
			return "", fmt.Errorf("proposal input %q escapes probe work directory", key)
		}
		return rendered, nil
	}
	if probe.isRelativePathInput(key, rendered) {
		return probe.safeInputPath(key, rendered)
	}
	return rendered, nil
}

func (probe probeMaterializer) isRelativePathInput(key, value string) bool {
	if value == "" || filepath.IsAbs(value) {
		return false
	}
	if strings.HasPrefix(value, "."+string(filepath.Separator)) || strings.HasPrefix(value, ".."+string(filepath.Separator)) {
		return true
	}
	key = strings.ToLower(strings.TrimSpace(key))
	switch key {
	case "source", "target", "input", "output":
		return true
	default:
		return strings.HasSuffix(key, "_path") || strings.HasSuffix(key, "path") ||
			strings.HasSuffix(key, "_file") || strings.HasSuffix(key, "file")
	}
}

func (probe probeMaterializer) safeInputPath(key, name string) (string, error) {
	clean := filepath.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("proposal input %q escapes probe work directory", key)
	}
	path := filepath.Join(probe.workDir, clean)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create proposal input directory: %w", err)
	}
	return path, nil
}

func (probe probeMaterializer) materializeFixture(fixture Fixture) (string, error) {
	if fixture.Input == "" {
		return "", fmt.Errorf("proposal fixture input is required")
	}
	path, err := probe.safeFixturePath(fixture.Filename)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(fixture.Content), 0o644); err != nil {
		return "", fmt.Errorf("write proposal fixture: %w", err)
	}
	return path, nil
}

func (probe probeMaterializer) safeFixturePath(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("proposal fixture filename is required")
	}
	if filepath.IsAbs(name) {
		return "", fmt.Errorf("proposal fixture filename must be relative")
	}
	clean := filepath.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("proposal fixture filename escapes probe work directory")
	}
	path := filepath.Join(probe.workDir, clean)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create proposal fixture directory: %w", err)
	}
	return path, nil
}

func (probe probeMaterializer) ensureVerifier(verifier core.Verifier) (core.Verifier, error) {
	for _, pkg := range probe.verifierPackages {
		if pkg.ID != verifier.ID {
			continue
		}
		verifier.ID = probe.finalVerifierID(pkg.ID)
		pkg.ID = verifier.ID
		if runtime.DefaultRegistry().Supports(verifier.ID) {
			return verifier, nil
		}
		if err := runtime.InstallVerifier(pkg); err != nil {
			return core.Verifier{}, err
		}
		return verifier, nil
	}
	if runtime.DefaultRegistry().Supports(verifier.ID) {
		return verifier, nil
	}
	return core.Verifier{}, fmt.Errorf("proposal verifier %q is not supported", verifier.ID)
}

func (probe probeMaterializer) finalVerifierID(localID string) string {
	hash := core.ShortHash(
		probe.proposalHash,
		strconv.Itoa(probe.candidateIndex),
		strconv.Itoa(probe.probePlanIndex),
		localID,
	)
	return "verifier_" + localID + "_" + hash
}

func (probe probeMaterializer) isWithinWorkDir(path string) bool {
	rel, err := filepath.Rel(probe.workDir, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
