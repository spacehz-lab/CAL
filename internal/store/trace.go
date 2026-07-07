package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spacehz-lab/cal/internal/model"
	"github.com/spacehz-lab/cal/pkg/jsonfile"
)

// ListTraces reads all stored trace records.
func (store *Store) ListTraces() ([]model.Trace, error) {
	dir := filepath.Join(store.root, tracesDir)
	entries, err := os.ReadDir(dir)
	if isNotExist(err) {
		return []model.Trace{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list traces: %w", err)
	}

	traces := make([]model.Trace, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		trace, ok, err := store.GetTrace(entry.Name())
		if err != nil {
			return nil, err
		}
		if ok {
			traces = append(traces, trace)
		}
	}
	sort.Slice(traces, func(i, j int) bool {
		return traces[i].ID < traces[j].ID
	})
	return traces, nil
}

// GetTrace reads one trace by id.
func (store *Store) GetTrace(id string) (model.Trace, bool, error) {
	if err := validateRecordID(id); err != nil {
		return model.Trace{}, false, err
	}

	var trace model.Trace
	if err := readJSON(store.tracePath(id), &trace); isNotExist(err) {
		return model.Trace{}, false, nil
	} else if err != nil {
		return model.Trace{}, false, err
	}
	if err := model.ValidateTrace(trace); err != nil {
		return model.Trace{}, false, err
	}
	return trace, true, nil
}

// SaveTrace writes one trace record.
func (store *Store) SaveTrace(trace *model.Trace) error {
	if trace == nil {
		return errNilRecord("trace")
	}
	if err := model.ValidateTrace(*trace); err != nil {
		return err
	}
	if err := validateRecordID(trace.ID); err != nil {
		return err
	}
	return jsonfile.WriteAtomic(store.tracePath(trace.ID), trace)
}

func (store *Store) tracePath(id string) string {
	return filepath.Join(store.root, tracesDir, id, traceFileName)
}
