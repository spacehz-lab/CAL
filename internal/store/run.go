package store

import (
	"github.com/spacehz-lab/cal/internal/core"
)

// PutRun writes one run record.
func (s *Store) PutRun(run core.Run) error {
	return putJSONRecord(s, runsDir, run.ID, run, core.ValidateRun)
}

// GetRun reads one run by id.
func (s *Store) GetRun(id string) (core.Run, bool, error) {
	return getJSONRecord(s, runsDir, id, core.ValidateRun)
}

// ListRuns reads all run records.
func (s *Store) ListRuns() ([]core.Run, error) {
	return listJSONRecords(s.home, runsDir, "runs", "run", core.ValidateRun, func(a, b core.Run) bool {
		return a.ID < b.ID
	})
}
