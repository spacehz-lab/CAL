package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

func readJSON(path string, target any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func isNotExist(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}
