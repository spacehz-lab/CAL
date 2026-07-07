package store

import "github.com/spacehz-lab/cal/internal/model"

// ListRuns reads all stored run records.
func (store *Store) ListRuns() ([]model.Run, error) {
	return listJSONRecords(store.root, runsDir, model.ValidateRun, func(a, b model.Run) bool {
		return a.ID < b.ID
	})
}

// GetRun reads one run by id.
func (store *Store) GetRun(id string) (model.Run, bool, error) {
	return getJSONRecord(store, runsDir, id, model.ValidateRun)
}

// SaveRun writes one run record.
func (store *Store) SaveRun(run *model.Run) error {
	if run == nil {
		return errNilRecord("run")
	}
	return saveJSONRecord(store, runsDir, run.ID, *run, model.ValidateRun)
}
