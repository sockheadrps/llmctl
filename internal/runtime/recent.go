package runtime

import (
	"encoding/json"
	"os"
	"time"

	"github.com/sockheadrps/llmctl/internal/models"
	"github.com/sockheadrps/llmctl/internal/util"
)

// loadRecent reads the rolling recent-runs history from path. A missing
// file is treated as an empty list, not an error.
func loadRecent(path string) ([]models.RecentRun, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var recent []models.RecentRun
	if err := json.Unmarshal(data, &recent); err != nil {
		return nil, err
	}
	return recent, nil
}

// saveRecent writes the recent-runs history to path.
func saveRecent(path string, recent []models.RecentRun) error {
	data, err := json.MarshalIndent(recent, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// recordRecent moves modelKey/profileKey to the front of the recent-runs
// history (deduplicating any earlier entry for the same pair) and trims
// the list to models.RecentLimit. Failures here are non-fatal to the
// caller's start — this is bookkeeping, not core functionality.
func recordRecent(modelKey, profileKey string) {
	path, err := util.RecentFile()
	if err != nil {
		return
	}

	recent, err := loadRecent(path)
	if err != nil {
		return
	}

	kept := make([]models.RecentRun, 0, len(recent)+1)
	kept = append(kept, models.RecentRun{
		ModelKey:   modelKey,
		ProfileKey: profileKey,
		RanAt:      time.Now().Unix(),
	})
	for _, r := range recent {
		if r.ModelKey == modelKey && r.ProfileKey == profileKey {
			continue
		}
		kept = append(kept, r)
	}
	if len(kept) > models.RecentLimit {
		kept = kept[:models.RecentLimit]
	}

	_ = saveRecent(path, kept)
}

// RecentRuns returns the rolling history of recently started model+profile
// pairs, most recent first.
func (mgr *Manager) RecentRuns() ([]models.RecentRun, error) {
	path, err := util.RecentFile()
	if err != nil {
		return nil, err
	}
	return loadRecent(path)
}
