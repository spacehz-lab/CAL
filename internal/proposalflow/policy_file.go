package proposalflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// LoadPolicyFile reads a complete Proposal policy file.
func LoadPolicyFile(path string) (Policy, error) {
	file, err := os.Open(path)
	if err != nil {
		return Policy{}, fmt.Errorf("open proposal policy: %w", err)
	}
	defer file.Close()

	var policy Policy
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&policy); err != nil {
		return Policy{}, fmt.Errorf("decode proposal policy: %w", err)
	}
	if err := ValidatePolicy(policy); err != nil {
		return Policy{}, err
	}
	return policy, nil
}

// EnsurePolicyFile writes the default complete policy when the file is missing.
func EnsurePolicyFile(path string) (Policy, error) {
	policy, err := LoadPolicyFile(path)
	if err == nil {
		return policy, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return Policy{}, err
	}
	policy = DefaultPolicy()
	if err := writePolicyFile(path, policy); err != nil {
		return Policy{}, err
	}
	return policy, nil
}

func writePolicyFile(path string, policy Policy) error {
	if err := ValidatePolicy(policy); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create proposal policy directory: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*.json")
	if err != nil {
		return fmt.Errorf("create temp proposal policy: %w", err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)

	encoder := json.NewEncoder(temp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(policy); err != nil {
		temp.Close()
		return fmt.Errorf("encode proposal policy: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close proposal policy: %w", err)
	}
	if err := os.Rename(tempName, path); err != nil {
		return fmt.Errorf("rename proposal policy: %w", err)
	}
	return nil
}
