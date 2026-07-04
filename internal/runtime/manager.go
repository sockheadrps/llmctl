// Package runtime tracks running llama-server instances across separate
// CLI invocations by persisting them to a JSON state file, reconciling
// against actual OS process liveness on every read.
package runtime

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/sockheadrps/llmctl/internal/config"
	"github.com/sockheadrps/llmctl/internal/models"
	"github.com/sockheadrps/llmctl/internal/process"
	"github.com/sockheadrps/llmctl/internal/util"
)

// Manager coordinates starting/stopping llama-server processes and
// persisting their state to disk.
type Manager struct {
	statePath string
}

// NewManager creates a Manager backed by the default state file location.
func NewManager() (*Manager, error) {
	statePath, err := util.StateFile()
	if err != nil {
		return nil, err
	}
	return &Manager{statePath: statePath}, nil
}

// LogPath returns the deterministic log file location for a model+profile
// pair, matching where Start writes its output. Exported so callers (e.g.
// the TUI's log viewer) can find a profile's log without having to run it
// first.
func LogPath(modelKey, profileKey string) (string, error) {
	logDir, err := util.LogDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(logDir, fmt.Sprintf("%s-%s.log", modelKey, profileKey)), nil
}

// List returns all instances currently believed to be running, pruning any
// whose PID is no longer alive.
func (mgr *Manager) List() ([]models.Running, error) {
	running, err := loadState(mgr.statePath)
	if err != nil {
		return nil, err
	}

	alive := make([]models.Running, 0, len(running))
	for _, r := range running {
		if process.IsAlive(r.PID) {
			alive = append(alive, r)
		}
	}

	if len(alive) != len(running) {
		if err := saveState(mgr.statePath, alive); err != nil {
			return nil, err
		}
	}

	return alive, nil
}

// Start launches modelKey/profileKey from cfg, records it in the state
// file, and returns the resulting Running entry. It refuses to start a
// model+profile pair that is already running.
func (mgr *Manager) Start(cfg *config.Config, modelKey, profileKey string) (models.Running, error) {
	m, p, err := cfg.FindProfile(modelKey, profileKey)
	if err != nil {
		return models.Running{}, err
	}

	running, err := mgr.List()
	if err != nil {
		return models.Running{}, err
	}
	for _, r := range running {
		if r.ModelKey == modelKey && r.ProfileKey == profileKey {
			return models.Running{}, fmt.Errorf("%s / %s is already running on port %d", m.Name, p.Name, r.Port)
		}
	}

	logPath, err := LogPath(modelKey, profileKey)
	if err != nil {
		return models.Running{}, err
	}

	pid, err := process.Start(cfg.LlamaServerBin, m, p, logPath)
	if err != nil {
		return models.Running{}, err
	}

	if err := awaitStableStart(pid, logPath); err != nil {
		return models.Running{}, fmt.Errorf("%s / %s failed to start: %w", m.Name, p.Name, err)
	}

	entry := models.Running{
		ModelKey:    modelKey,
		ModelName:   m.Name,
		ProfileKey:  profileKey,
		ProfileName: p.Name,
		Port:        p.Port,
		PID:         pid,
		LogFile:     logPath,
		StartedAt:   time.Now().Unix(),
	}

	running = append(running, entry)
	if err := saveState(mgr.statePath, running); err != nil {
		return models.Running{}, err
	}

	recordRecent(modelKey, profileKey)

	return entry, nil
}

// startGracePeriod is how long Start waits to confirm the process didn't
// die immediately (e.g. from a bad CLI argument) before declaring success.
// It doesn't wait for the model to finish loading, just for the process to
// survive past its own startup argument parsing.
const startGracePeriod = 2500 * time.Millisecond

// awaitStableStart polls pid for startGracePeriod. If the process exits
// during that window, it returns an error containing the tail of its log
// so the caller can see why (e.g. an invalid flag or missing model file).
func awaitStableStart(pid int, logPath string) error {
	deadline := time.Now().Add(startGracePeriod)
	for time.Now().Before(deadline) {
		if !process.IsAlive(pid) {
			if tail, err := process.TailLog(logPath, 8); err == nil && tail != "" {
				return fmt.Errorf("process exited immediately:\n%s", tail)
			}
			return fmt.Errorf("process exited immediately")
		}
		time.Sleep(150 * time.Millisecond)
	}
	return nil
}

// Stop terminates the running instance for modelKey/profileKey and removes
// it from the state file.
func (mgr *Manager) Stop(modelKey, profileKey string) error {
	running, err := mgr.List()
	if err != nil {
		return err
	}

	kept := make([]models.Running, 0, len(running))
	var target *models.Running
	for _, r := range running {
		if r.ModelKey == modelKey && r.ProfileKey == profileKey {
			r := r
			target = &r
			continue
		}
		kept = append(kept, r)
	}

	if target == nil {
		return fmt.Errorf("%s / %s is not running", modelKey, profileKey)
	}

	if err := process.Stop(target.PID); err != nil {
		return err
	}

	return saveState(mgr.statePath, kept)
}

// Find returns the Running entry for modelKey/profileKey, if any.
func (mgr *Manager) Find(modelKey, profileKey string) (models.Running, bool, error) {
	running, err := mgr.List()
	if err != nil {
		return models.Running{}, false, err
	}
	for _, r := range running {
		if r.ModelKey == modelKey && r.ProfileKey == profileKey {
			return r, true, nil
		}
	}
	return models.Running{}, false, nil
}
