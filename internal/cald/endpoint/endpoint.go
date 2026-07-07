package endpoint

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spacehz-lab/cal/pkg/jsonfile"
)

const (
	dirName  = "cald"
	fileName = "endpoint.json"
)

// Record is the local cald endpoint metadata read by daemon clients.
type Record struct {
	BaseURL   string `json:"base_url"`
	PID       int    `json:"pid,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

// Path returns the endpoint metadata path under one CAL home.
func Path(home string) string {
	return filepath.Join(home, dirName, fileName)
}

// Read loads the endpoint metadata record.
func Read(home string) (Record, bool, error) {
	path, err := checkedPath(home)
	if err != nil {
		return Record{}, false, err
	}

	var record Record
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return Record{}, false, nil
	}
	if err != nil {
		return Record{}, false, fmt.Errorf("open endpoint: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&record); err != nil {
		return Record{}, false, fmt.Errorf("decode endpoint: %w", err)
	}
	if err := record.Validate(); err != nil {
		return Record{}, false, err
	}
	return record, true, nil
}

// Write validates and writes the endpoint metadata record atomically.
func Write(home string, record *Record) error {
	path, err := checkedPath(home)
	if err != nil {
		return err
	}
	if record == nil {
		return fmt.Errorf("endpoint record is required")
	}
	if err := record.Validate(); err != nil {
		return err
	}
	return jsonfile.WriteAtomic(path, record)
}

// Remove deletes the endpoint metadata record when it exists.
func Remove(home string) error {
	path, err := checkedPath(home)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove endpoint: %w", err)
	}
	return nil
}

// Validate checks the endpoint metadata contract.
func (record *Record) Validate() error {
	if record == nil {
		return fmt.Errorf("endpoint record is required")
	}
	if strings.TrimSpace(record.BaseURL) == "" {
		return fmt.Errorf("endpoint base url is required")
	}
	return nil
}

func checkedPath(home string) (string, error) {
	if strings.TrimSpace(home) == "" {
		return "", fmt.Errorf("cal home is required")
	}
	return Path(home), nil
}
