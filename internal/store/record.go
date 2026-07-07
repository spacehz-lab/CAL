package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spacehz-lab/cal/pkg/jsonfile"
)

func (store *Store) recordPath(dirName, id string) (string, error) {
	if err := validateRecordID(id); err != nil {
		return "", err
	}
	return filepath.Join(store.root, dirName, id+jsonExt), nil
}

func listJSONRecords[T any](root, dirName string, validate func(T) error, less func(T, T) bool) ([]T, error) {
	dir := filepath.Join(root, dirName)
	entries, err := os.ReadDir(dir)
	if isNotExist(err) {
		return []T{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", dir, err)
	}

	records := make([]T, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != jsonExt {
			continue
		}

		var record T
		path := filepath.Join(dir, entry.Name())
		if err := readJSON(path, &record); err != nil {
			return nil, err
		}
		if err := validate(record); err != nil {
			return nil, fmt.Errorf("validate %s: %w", path, err)
		}
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		return less(records[i], records[j])
	})
	return records, nil
}

func getJSONRecord[T any](store *Store, dirName, id string, validate func(T) error) (T, bool, error) {
	path, err := store.recordPath(dirName, id)
	if err != nil {
		var zero T
		return zero, false, err
	}

	var record T
	if err := readJSON(path, &record); isNotExist(err) {
		return record, false, nil
	} else if err != nil {
		return record, false, err
	}
	if err := validate(record); err != nil {
		return record, false, err
	}
	return record, true, nil
}

func saveJSONRecord[T any](store *Store, dirName, id string, record T, validate func(T) error) error {
	if err := validate(record); err != nil {
		return err
	}
	path, err := store.recordPath(dirName, id)
	if err != nil {
		return err
	}
	return jsonfile.WriteAtomic(path, record)
}

func validateRecordID(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("record id is required")
	}
	if id == "." || id == ".." || strings.ContainsAny(id, `/\`) {
		return fmt.Errorf("record id %q is not path safe", id)
	}
	return nil
}

func errNilRecord(name string) error {
	return fmt.Errorf("%s record is nil", name)
}
