package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sockheadrps/llmctl/internal/config"
	"github.com/sockheadrps/llmctl/internal/models"
	tui_picker "github.com/sockheadrps/llmctl/internal/tui/picker"
)

func (m Model) openPicker() (tea.Model, tea.Cmd) {
	state, err := tui_picker.Open(m.cfg)
	if err != nil {
		m.setError(err, "")
		return m, nil
	}
	m.picker = state
	m.screen = screenPickModel
	m.clearError()
	return m, nil
}

func (m Model) updatePicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.picker.Update(msg) {
	case tui_picker.ActionBack:
		m.screen = screenMain
		return m, nil
	case tui_picker.ActionImport:
		return m.importSelectedModel()
	default:
		return m, nil
	}
}

// importSelectedModel adds the picked GGUF file as a model, or reuses an
// existing hidden model entry instead of creating a duplicate.
func (m Model) importSelectedModel() (tea.Model, tea.Cmd) {
	if len(m.picker.Files) == 0 {
		return m, nil
	}
	path := m.picker.Files[m.picker.Cursor]

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

// tuiDefaultProfile wraps models.DefaultProfile so the TUI side can still
// apply a port suggestion that avoids collisions with existing profiles in
// this config.
func tuiDefaultProfile(cfg *config.Config) models.Profile {
	p := models.DefaultProfile()
	p.Port = suggestPort(cfg)
	return p
}
