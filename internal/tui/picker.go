package tui

import (
	"fmt"
	"sort"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sockheadrps/llmctl/internal/config"
	"github.com/sockheadrps/llmctl/internal/models"
)

// pickerState backs the "Add Model" screen: a list of .gguf files found
// under the config's models_dirs that aren't already registered. unreadable
// notes which configured directories couldn't be scanned (missing, no
// permission, …) — a per-directory problem, not fatal to the whole screen.
type pickerState struct {
	files      []string
	cursor     int
	err        error
	unreadable []string
}

// openPicker scans every configured models directory for importable GGUF
// files and switches to the model-picker screen. A directory that can't be
// scanned (not mounted yet, typo, …) is skipped rather than failing the
// whole picker — its problem is surfaced via pickerState.unreadable instead.
func (m Model) openPicker() (tea.Model, tea.Cmd) {
	dirs, err := m.cfg.ResolvedModelsDirs()
	if err != nil {
		m.setError(err, "")
		return m, nil
	}

	// Only paths belonging to models that are actually visible (i.e. have
	// at least one profile) count as "already imported". A model hidden
	// for having zero profiles should be offered again so re-selecting it
	// can bring it back rather than dead-ending at "no new files found".
	used := make(map[string]bool, len(m.cfg.Models))
	for _, mdl := range m.cfg.Models {
		if len(mdl.Profiles) > 0 {
			used[mdl.Path] = true
		}
	}

	var files, unreadable []string
	for _, dir := range dirs {
		found, err := models.ScanGGUF(dir)
		if err != nil {
			unreadable = append(unreadable, dir)
			continue
		}
		for _, f := range found {
			if !used[f] {
				files = append(files, f)
			}
		}
	}
	sort.Strings(files)

	if len(dirs) == 0 {
		m.picker = pickerState{err: fmt.Errorf("no model directories configured — add one from the Model Directories screen")}
	} else {
		m.picker = pickerState{files: files, unreadable: unreadable}
	}
	m.screen = screenPickModel
	m.clearError()
	return m, nil
}

func (m Model) updatePicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.screen = screenMain
		return m, nil

	case "up", "k":
		if m.picker.cursor > 0 {
			m.picker.cursor--
		}
		return m, nil

	case "down", "j":
		if m.picker.cursor < len(m.picker.files)-1 {
			m.picker.cursor++
		}
		return m, nil

	case "enter":
		return m.importSelectedModel()
	}
	return m, nil
}

// importSelectedModel adds the picked GGUF file as a model, or — if it
// matches an existing model currently hidden for having no profiles —
// reuses that entry instead of creating a duplicate. Either way the result
// gets a "default" profile so it has something to run and isn't hidden
// again by buildRows.
func (m Model) importSelectedModel() (tea.Model, tea.Cmd) {
	if len(m.picker.files) == 0 {
		return m, nil
	}
	path := m.picker.files[m.picker.cursor]

	key := models.ModelKeyByPath(m.cfg.Models, path)
	if key == "" {
		key = models.UniqueModelKey(m.cfg.Models, models.ModelKeyFromPath(path))
		m.cfg.Models[key] = models.Model{
			Key:      key,
			Name:     models.ModelNameFromPath(path),
			Path:     path,
			Profiles: map[string]models.Profile{},
		}
	}

	mdl := m.cfg.Models[key]
	if len(mdl.Profiles) == 0 {
		if mdl.Profiles == nil {
			mdl.Profiles = map[string]models.Profile{}
		}
		mdl.Profiles["default"] = tuiDefaultProfile(m.cfg)
		m.cfg.Models[key] = mdl
	}

	if err := m.saveConfig(); err != nil {
		m.setError(err, "")
		return m, nil
	}

	// Expand it in the tree so the profile you just landed with (a fresh
	// default one, or an existing model's) is immediately visible instead
	// of hiding behind another "enter to expand" step.
	m.expandedModelKey = key
	m.rebuildRows()
	m.cursor = indexOfModelRow(m.rows, key)
	m.screen = screenMain
	m.clearError()
	return m, nil
}

func indexOfModelRow(rows []row, modelKey string) int {
	for i, r := range rows {
		if r.kind == rowModel && r.modelKey == modelKey {
			return i
		}
	}
	return 0
}

// tuiDefaultProfile wraps models.DefaultProfile so the TUI side can
// still apply a port suggestion that avoids collisions with existing
// profiles in this config. (Section 3 extraction kept DefaultProfile
// pure; the TUI still owns port-collision awareness.)
func tuiDefaultProfile(cfg *config.Config) models.Profile {
	p := models.DefaultProfile()
	p.Port = suggestPort(cfg)
	return p
}

