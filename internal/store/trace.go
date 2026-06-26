package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	caltrace "github.com/spacehz-lab/cal/internal/trace"
)

// PutTrace writes one discovery trace record.
func (s *Store) PutTrace(record caltrace.Trace) error {
	if err := caltrace.Validate(record); err != nil {
		return err
	}
	if err := validateRecordID(record.ID); err != nil {
		return err
	}
	return writeJSONAtomic(s.tracePath(record.ID), record)
}

// GetTrace reads one discovery trace by id.
func (s *Store) GetTrace(id string) (caltrace.Trace, bool, error) {
	if err := validateRecordID(id); err != nil {
		return caltrace.Trace{}, false, err
	}

	var trace caltrace.Trace
	if err := readJSON(s.tracePath(id), &trace); isNotExist(err) {
		return caltrace.Trace{}, false, nil
	} else if err != nil {
		return caltrace.Trace{}, false, err
	}
	if err := caltrace.Validate(trace); err != nil {
		return caltrace.Trace{}, false, err
	}
	return trace, true, nil
}

// ListTraces reads all discovery trace records.
func (s *Store) ListTraces() ([]caltrace.Trace, error) {
	dir := filepath.Join(s.home, discoveryDir)
	entries, err := os.ReadDir(dir)
	if isNotExist(err) {
		return []caltrace.Trace{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list traces: %w", err)
	}

	traces := make([]caltrace.Trace, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		trace, ok, err := s.GetTrace(entry.Name())
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

func (s *Store) tracePath(id string) string {
	return filepath.Join(s.home, discoveryDir, id, "trace.json")
}

// PrepareTraceProbeDir creates and returns a work directory for one trace probe.
func (s *Store) PrepareTraceProbeDir(traceID string, candidateIndex int) (string, error) {
	if err := validateRecordID(traceID); err != nil {
		return "", err
	}
	if candidateIndex < 0 {
		return "", fmt.Errorf("candidate index must be non-negative")
	}
	dir := filepath.Join(s.home, discoveryDir, traceID, "probes", fmt.Sprintf("%d", candidateIndex))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create trace probe directory: %w", err)
	}
	return dir, nil
}
